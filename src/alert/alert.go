// Package alert is the sink for things that should never have happened: panics,
// 5xx responses, failed background jobs, and crashes reported by the app. It
// posts them to a Discord channel with enough context to start tracing.
//
// The API does not call this package directly on the request path — it publishes
// an alert job to the bus (see events.EmitAlert) and cmd/worker delivers it, so a
// slow or rate-limited Discord can never slow down a request. The one exception
// is the bus being down, in which case events falls back to calling Send here
// inline: a dead queue is exactly when an alert matters most.
package alert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"coachwise/src/logger"
)

// Discord's limits, which are not suggestions: an embed description caps at 4096
// characters and a whole embed at 6000. A Go stack trace blows through both, so
// it is truncated rather than silently rejected with a 400.
const (
	maxStack       = 1500
	maxFieldValue  = 1000
	queueSize      = 256
	minGapBetween  = 1500 * time.Millisecond // ~40/min, under Discord's ~30/min burst ceiling
	dedupeCooldown = 5 * time.Minute
)

// Kind decides the colour of the embed and how loud the alert reads.
type Kind string

const (
	KindPanic  Kind = "panic"  // an unrecovered panic — always a bug
	KindServer Kind = "5xx"    // a request that left as a server error
	KindWorker Kind = "worker" // a background job that failed
	KindClient Kind = "app"    // a crash reported by the frontend
)

var colors = map[Kind]int{
	KindPanic:  0xE03131, // red
	KindServer: 0xF08C00, // orange
	KindWorker: 0xE8590C, // dark orange
	KindClient: 0x1971C2, // blue — someone else's browser, not the server falling over
}

// Event is one thing that went wrong. Fields are ordered, because the order they
// were written in is the order that is useful to read.
type Event struct {
	Kind        Kind      `json:"kind"`
	Title       string    `json:"title"`  // one line: what broke
	Detail      string    `json:"detail"` // the error/panic message
	Stack       string    `json:"stack"`  // stack trace, already truncated by the producer where possible
	Fields      []Field   `json:"fields"` // route, user, request id, body...
	Fingerprint string    `json:"fingerprint"`
	At          time.Time `json:"at"`
}

type Field struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

var (
	webhook string
	env     string

	queue chan Event
	once  sync.Once

	mu   sync.Mutex
	seen = map[string]*repeat{}
)

// repeat tracks how often a fingerprint has fired inside the cooldown, so a crash
// loop becomes one message saying "×412" instead of 412 messages saying nothing.
type repeat struct {
	last  time.Time
	count int
}

// Init wires the sink. An empty webhook is not an error: alerts are logged
// instead, which is what dev and the test suite want.
func Init(webhookURL, environment string) {
	webhook = webhookURL
	env = environment
	if webhook == "" {
		logger.Info("alert: no discord webhook configured — alerts will be logged only")
	}
	once.Do(func() {
		queue = make(chan Event, queueSize)
		go run()
	})
}

// Send hands an event to the sender. It never blocks and never panics: if the
// queue is full the event is dropped with a log line, because back-pressure from
// an alerting system must not become an outage of its own.
func Send(e Event) {
	if queue == nil {
		logger.Errorf("alert (sink not initialised): %s — %s", e.Title, e.Detail)
		return
	}
	if e.At.IsZero() {
		e.At = time.Now()
	}
	select {
	case queue <- e:
	default:
		logger.Errorf("alert: queue full, dropped: %s", e.Title)
	}
}

func run() {
	var lastSent time.Time
	for e := range queue {
		count, ok := admit(e)
		if !ok {
			continue // still inside the cooldown for this fingerprint
		}
		// Space the posts out; Discord 429s a burst and we would lose the tail.
		if gap := time.Since(lastSent); gap < minGapBetween {
			time.Sleep(minGapBetween - gap)
		}
		post(e, count)
		lastSent = time.Now()
	}
}

// admit applies the dedupe window. The first occurrence goes out immediately; the
// rest are counted, and the count rides along on the next message after the
// window closes — so a repeating failure reports its true volume.
func admit(e Event) (int, bool) {
	if e.Fingerprint == "" {
		return 1, true
	}
	mu.Lock()
	defer mu.Unlock()

	r, ok := seen[e.Fingerprint]
	now := time.Now()
	if !ok || now.Sub(r.last) > dedupeCooldown {
		seen[e.Fingerprint] = &repeat{last: now, count: 0}
		// Cheap sweep so the map can't grow without bound.
		if len(seen) > 500 {
			for k, v := range seen {
				if now.Sub(v.last) > dedupeCooldown {
					delete(seen, k)
				}
			}
		}
		if ok && r.count > 0 {
			return r.count + 1, true // report what we suppressed
		}
		return 1, true
	}
	r.count++
	return 0, false
}

func post(e Event, count int) {
	title := e.Title
	if count > 1 {
		title = fmt.Sprintf("%s  (×%d in the last %s)", title, count, dedupeCooldown)
	}

	if webhook == "" {
		logger.Errorf("[alert:%s] %s\n%s\n%s", e.Kind, title, e.Detail, e.Stack)
		return
	}

	fields := make([]map[string]any, 0, len(e.Fields)+1)
	for _, f := range e.Fields {
		if f.Value == "" {
			continue
		}
		fields = append(fields, map[string]any{
			"name":   f.Name,
			"value":  truncate(f.Value, maxFieldValue),
			"inline": f.Inline,
		})
	}

	desc := e.Detail
	if e.Stack != "" {
		desc += "\n```\n" + truncate(e.Stack, maxStack) + "\n```"
	}

	body, err := json.Marshal(map[string]any{
		"embeds": []map[string]any{{
			"title":       truncate(title, 250),
			"description": truncate(desc, 4000),
			"color":       colors[e.Kind],
			"fields":      fields,
			"footer":      map[string]any{"text": fmt.Sprintf("%s · %s", e.Kind, env)},
			"timestamp":   e.At.UTC().Format(time.RFC3339),
		}},
	})
	if err != nil {
		logger.Errorf("alert: marshal: %v", err)
		return
	}

	// One retry, and only for a 429 — Discord tells us how long to wait. Anything
	// else is logged and dropped: alerting must not turn into a retry storm.
	for attempt := 0; attempt < 2; attempt++ {
		resp, err := http.Post(webhook, "application/json", bytes.NewReader(body))
		if err != nil {
			logger.Errorf("alert: post failed: %v", err)
			return
		}
		status := resp.StatusCode
		retryAfter := resp.Header.Get("Retry-After")
		resp.Body.Close()

		if status == http.StatusTooManyRequests && attempt == 0 {
			wait := 2 * time.Second
			if s, err := strconv.ParseFloat(retryAfter, 64); err == nil && s > 0 && s < 30 {
				wait = time.Duration(s * float64(time.Second))
			}
			time.Sleep(wait)
			continue
		}
		if status >= 300 {
			logger.Errorf("alert: discord returned %d", status)
		}
		return
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n… truncated"
}

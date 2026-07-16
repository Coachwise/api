package events

import (
	"context"
	"encoding/json"
	"time"

	"coachwise/src/app/models"
	"coachwise/src/logger"
	"coachwise/src/utils"

	"github.com/nats-io/nats.go"
)

// SubjectSupportPing carries SupportPing — a heads-up that a ticket needs a look.
// The API publishes; cmd/worker delivers it to Discord, so a slow or rate-limited
// webhook is never latency on the user's request.
const SubjectSupportPing = "events.support_ping"

// One worker posts each ping; the queue group makes that true across any number
// of worker processes.
const supportQueueGroup = "support-workers"

// supportWebhook is the Discord destination, set by InitSupport in each process.
// Both the API (inline fallback) and the worker (consumer) need it.
var supportWebhook string

// InitSupport wires the support-ping webhook. An empty webhook is not an error:
// PostDiscordWebhook logs the ping instead, which is how dev works.
func InitSupport(webhook string) { supportWebhook = webhook }

// SupportPing is intentionally just the finished Discord message, so the event
// stays independent of how the ping is worded or what a ticket looks like.
type SupportPing struct {
	Content string `json:"content"`
}

// EmitSupportPing queues a heads-up for Discord delivery by the worker. When the
// bus is down it delivers inline instead. Never blocks the caller either way —
// PostDiscordWebhook is async.
func EmitSupportPing(content string) {
	if c := getConn(); c != nil && c.IsConnected() {
		if b, err := json.Marshal(SupportPing{Content: content}); err == nil {
			if err := c.Publish(SubjectSupportPing, b); err == nil {
				return
			} else {
				logger.Errorf("events: publish support ping failed, sending inline: %v", err)
			}
		}
	}
	utils.PostDiscordWebhook(supportWebhook, content)
}

// StartSupportConsumer subscribes the worker to support pings. No-op when the bus
// is disabled — producers have already delivered inline in that case.
func StartSupportConsumer() {
	c := getConn()
	if c == nil {
		logger.Info("events: support consumer not started (bus disabled)")
		return
	}
	_, err := c.QueueSubscribe(SubjectSupportPing, supportQueueGroup, func(m *nats.Msg) {
		var p SupportPing
		if err := json.Unmarshal(m.Data, &p); err != nil {
			logger.Errorf("events: bad support ping: %v", err)
			return
		}
		utils.PostDiscordWebhook(supportWebhook, p.Content)
	})
	if err != nil {
		logger.Errorf("events: support subscribe failed: %v", err)
		return
	}
	logger.Info("events: support consumer subscribed")
}

// StartSupportDeliveryLoop is the bridge the whole feature turns on. Admins answer
// tickets in the admin panel, which writes straight to Postgres and cannot reach
// the event bus — so nothing would ever tell the user. This loop polls for the
// admin replies not yet pushed, claims them atomically, and for each one raises
// an in-app notification and a "support" refetch signal, exactly as if the reply
// had come through the API. Runs in cmd/worker.
func StartSupportDeliveryLoop(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			deliverSupportReplies()
		}
	}()
	logger.Infof("events: support delivery loop running (every %s)", interval)
}

func deliverSupportReplies() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	replies, err := models.ClaimUndeliveredReplies(ctx)
	if err != nil {
		logger.Errorf("events: claim support replies: %v", err)
		return
	}

	entityType := "support_ticket"
	for _, r := range replies {
		ticketID := r.TicketID
		EmitNotification(r.UserID, nil, models.NotifSupportReply, &entityType, &ticketID,
			map[string]string{"preview": utils.TruncateRunes(r.Body, 140)})
		// Poke the open ticket view too, so an app sitting on the thread updates live.
		EmitSignal(r.UserID, "support")
	}
}

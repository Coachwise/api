package views

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"coachwise/src/alert"
	"coachwise/src/errcode"
	"coachwise/src/events"
	"coachwise/src/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// maxBodyCapture bounds what we read back for an alert. A 10MB upload must not be
// buffered into memory just so it can be thrown away by the redactor.
const maxBodyCapture = 4 << 10 // 4KB

// RequestID stamps every request with an id, echoes it in the response header,
// and puts it on the context. It is the thread that ties three things together:
// the log line, the Discord alert, and — because the app reads the header off a
// failed response and sends it back with its own report — the crash the user saw.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// Alerts recovers panics and reports server errors.
//
// It replaces gin's own Recovery middleware rather than sitting next to it: gin's
// recovers the panic first, writes a bare 500, and we would never see it.
func Alerts() gin.HandlerFunc {
	return func(c *gin.Context) {
		// The handler consumes the body, so capture it now or there is nothing
		// left to report. Bounded, and put straight back for the handler to read.
		var body []byte
		if c.Request.Body != nil && c.Request.ContentLength > 0 && c.Request.ContentLength <= maxBodyCapture {
			if b, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBodyCapture)); err == nil {
				body = b
				c.Request.Body = io.NopCloser(bytes.NewReader(b))
			}
		}

		started := time.Now()

		defer func() {
			if rec := recover(); rec != nil {
				stack := string(debug.Stack())
				logger.Errorf("panic on %s %s: %v\n%s", c.Request.Method, c.Request.URL.Path, rec, stack)

				events.EmitAlert(alert.Event{
					Kind:        alert.KindPanic,
					Title:       fmt.Sprintf("panic: %s %s", c.Request.Method, routeOf(c)),
					Detail:      fmt.Sprint(rec),
					Stack:       stack,
					Fingerprint: fingerprint("panic", routeOf(c), topFrame(stack)),
					Fields:      requestFields(c, body, started),
					At:          time.Now(),
				})

				// The client still gets a coded response, not a dropped connection.
				if !c.Writer.Written() {
					errcode.Abort(c, errcode.CodeUnknown)
				} else {
					c.Abort()
				}
			}
		}()

		c.Next()

		// A 5xx that did not panic: a handler caught an error and gave up. That is
		// still our bug, and it is the case gin's Recovery never sees.
		if status := c.Writer.Status(); status >= http.StatusInternalServerError {
			detail := "handler returned a server error"
			if len(c.Errors) > 0 {
				detail = c.Errors.String()
			}
			events.EmitAlert(alert.Event{
				Kind:        alert.KindServer,
				Title:       fmt.Sprintf("%d: %s %s", status, c.Request.Method, routeOf(c)),
				Detail:      detail,
				Fingerprint: fingerprint("5xx", routeOf(c), detail),
				Fields:      requestFields(c, body, started),
				At:          time.Now(),
			})
		}
	}
}

// requestFields is everything needed to start tracing: who, where, with what.
func requestFields(c *gin.Context, body []byte, started time.Time) []alert.Field {
	userID := ""
	if v, ok := c.Get("id"); ok {
		userID = fmt.Sprint(v)
	}
	reqID, _ := c.Get("request_id")

	return []alert.Field{
		{Name: "Request", Value: fmt.Sprintf("`%s %s`", c.Request.Method, c.Request.URL.Path), Inline: false},
		{Name: "Request ID", Value: fmt.Sprint(reqID), Inline: true},
		{Name: "User", Value: orNone(userID), Inline: true},
		{Name: "Status", Value: fmt.Sprint(c.Writer.Status()), Inline: true},
		{Name: "Took", Value: time.Since(started).Round(time.Millisecond).String(), Inline: true},
		{Name: "Client IP", Value: alert.Mask(c.ClientIP()), Inline: true},
		{Name: "Query", Value: orNone(c.Request.URL.RawQuery), Inline: false},
		// Redacted: a request body here can contain a live OTP, and this channel
		// is readable by the whole org and never expires. See alert/redact.go.
		{Name: "Body", Value: orNone(alert.RedactBody(c.ContentType(), body)), Inline: false},
		{Name: "Headers", Value: orNone(alert.RedactHeaders(c.GetHeader)), Inline: false},
	}
}

// routeOf prefers the matched route pattern (/users/:id) over the concrete path
// (/users/9f3e…), so every id of the same bug lands on one fingerprint.
func routeOf(c *gin.Context) string {
	if r := c.FullPath(); r != "" {
		return r
	}
	return c.Request.URL.Path
}

// fingerprint groups the same bug together across requests, so a crash loop is
// one Discord message with a count rather than a thousand identical ones.
func fingerprint(parts ...string) string {
	h := sha1.Sum([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(h[:8])
}

// topFrame is the first line of the stack below the runtime's own panic frames —
// enough to tell two different panics on the same route apart.
func topFrame(stack string) string {
	for _, line := range strings.Split(stack, "\n") {
		if strings.Contains(line, "coachwise/src/") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func orNone(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

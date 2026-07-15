package views

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"coachwise/src/alert"
	"coachwise/src/app/auth"
	"coachwise/src/events"

	"github.com/gin-gonic/gin"
)

// The app posts its crashes here and the API forwards them to Discord. The
// webhook URL stays on the server: shipped in the frontend bundle it would be a
// public URL anyone could extract and flood the channel with.
//
// Unauthenticated on purpose — the crash we most need to see is the one on the
// login screen, before anyone has a token. It is rate-limited per IP instead, and
// the payload is capped and redacted like any other alert.
func telemetryGroup(router *gin.Engine) {
	g := router.Group("telemetry")
	g.Use(rateLimit(20, time.Minute))

	g.POST("/error", func(c *gin.Context) {
		form := new(ClientErrorForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}

		// If a token came along, name the user; if not, the report still stands.
		userID := "—"
		if claims, err := auth.VerifyToken(bearer(c)); err == nil && claims != nil {
			userID = claims.ID
		}

		events.EmitAlert(alert.Event{
			Kind:   alert.KindClient,
			Title:  fmt.Sprintf("app: %s", truncateLine(form.Message, 120)),
			Detail: form.Message,
			Stack:  form.Stack,
			// Grouped by message + where it happened, NOT by user: one bug hitting
			// fifty people should be one alert that says fifty, not fifty alerts.
			Fingerprint: fingerprint("app", form.Kind, form.Message, form.View),
			Fields: []alert.Field{
				{Name: "Where", Value: orNone(form.View), Inline: true},
				{Name: "Kind", Value: orNone(form.Kind), Inline: true},
				{Name: "User", Value: userID, Inline: true},
				{Name: "App", Value: fmt.Sprintf("%s · %s", orNone(form.Version), orNone(form.Platform)), Inline: true},
				{Name: "Language", Value: orNone(form.Language), Inline: true},
				// Set when the app is reporting a failed API call: it reads the id
				// off the failing response, so this ties the two halves together.
				{Name: "API Request ID", Value: orNone(form.RequestID), Inline: true},
				{Name: "URL", Value: orNone(form.URL), Inline: false},
				{Name: "User Agent", Value: orNone(c.GetHeader("User-Agent")), Inline: false},
			},
			At: time.Now(),
		})

		// Deliberately 202 and empty: the app must never care what happened here.
		c.Status(http.StatusAccepted)
	})
}

func bearer(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if after, ok := strings.CutPrefix(h, "Bearer "); ok {
		return after
	}
	return ""
}

func truncateLine(s string, n int) string {
	s = strings.SplitN(s, "\n", 2)[0]
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

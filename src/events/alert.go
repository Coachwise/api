package events

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"coachwise/src/alert"
	"coachwise/src/logger"

	"github.com/nats-io/nats.go"
)

// SubjectAlert carries alert.Event — panics, 5xx responses, failed jobs and app
// crashes. The API publishes; cmd/worker delivers them to Discord, so a slow or
// rate-limited webhook can never show up as latency on somebody's request.
const SubjectAlert = "events.alert"

// One worker posts each alert; the queue group makes that true no matter how many
// worker processes are running.
const alertQueueGroup = "alert-workers"

// EmitAlert queues an event for delivery. When the bus is down it delivers inline
// instead — a dead queue is an outage, and an outage is precisely when the alert
// has to arrive. Never blocks the caller either way (alert.Send is async).
func EmitAlert(e alert.Event) {
	if c := getConn(); c != nil && c.IsConnected() {
		if b, err := json.Marshal(e); err == nil {
			if err := c.Publish(SubjectAlert, b); err == nil {
				return
			} else {
				logger.Errorf("events: publish alert failed, sending inline: %v", err)
			}
		}
	}
	alert.Send(e)
}

// StartAlertConsumer subscribes the worker to alert events. No-op when the bus is
// disabled — in that case producers have already delivered inline.
func StartAlertConsumer() {
	c := getConn()
	if c == nil {
		logger.Info("events: alert consumer not started (bus disabled)")
		return
	}
	_, err := c.QueueSubscribe(SubjectAlert, alertQueueGroup, func(m *nats.Msg) {
		var e alert.Event
		if err := json.Unmarshal(m.Data, &e); err != nil {
			logger.Errorf("events: bad alert job: %v", err)
			return
		}
		alert.Send(e)
	})
	if err != nil {
		logger.Errorf("events: alert subscribe failed: %v", err)
		return
	}
	logger.Info("events: alert consumer subscribed")
}

// reportJobFailure alerts on a background job that failed after being dequeued —
// a notification that would not persist, an SMS the provider rejected. These are
// worker bugs the user never sees surfaced any other way. Grouped by job + stage
// so one broken provider is one alert, not one per message.
func reportJobFailure(job, stage string, err error) {
	h := sha1.Sum([]byte("worker|" + job + "|" + stage))
	EmitAlert(alert.Event{
		Kind:        alert.KindWorker,
		Title:       fmt.Sprintf("worker: %s job failed (%s)", job, stage),
		Detail:      err.Error(),
		Fingerprint: hex.EncodeToString(h[:8]),
		Fields: []alert.Field{
			{Name: "Job", Value: job, Inline: true},
			{Name: "Stage", Value: stage, Inline: true},
		},
	})
}

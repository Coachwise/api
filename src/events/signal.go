package events

import (
	"encoding/json"
	"coachwise/src/logger"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// SubjectSignal carries RefetchSignal messages — tiny "your <topic> changed, go
// refetch it" pokes for connected clients. No payload/content, just a hint.
const SubjectSignal = "events.signal"

// RefetchSignal tells one user's live clients which sections to reload.
type RefetchSignal struct {
	UserID uuid.UUID `json:"user_id"`
	Topics []string  `json:"topics"`
}

// EmitSignal publishes a refetch hint for a user (e.g. "notifications",
// "messages", "connections"). Best-effort; a no-op when the bus is down (clients
// still get the data on their next normal fetch).
func EmitSignal(userID uuid.UUID, topics ...string) {
	c := getConn()
	if c == nil || !c.IsConnected() || len(topics) == 0 {
		return
	}
	payload, err := json.Marshal(RefetchSignal{UserID: userID, Topics: topics})
	if err != nil {
		return
	}
	if err := c.Publish(SubjectSignal, payload); err != nil {
		logger.Errorf("events: publish signal: %v", err)
	}
}

// SubscribeSignals delivers every refetch signal to handler (fan-out — no queue
// group, so every API instance sees it and pushes to its own connected clients).
func SubscribeSignals(handler func(RefetchSignal)) {
	c := getConn()
	if c == nil {
		logger.Info("events: signal subscribe skipped (bus disabled)")
		return
	}
	_, err := c.Subscribe(SubjectSignal, func(m *nats.Msg) {
		var sig RefetchSignal
		if err := json.Unmarshal(m.Data, &sig); err != nil {
			return
		}
		handler(sig)
	})
	if err != nil {
		logger.Errorf("events: signal subscribe failed: %v", err)
		return
	}
	logger.Info("events: signal subscriber ready")
}

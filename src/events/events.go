// Package events is a thin NATS-backed event bus. Producers publish events;
// consumers (in-process and/or dedicated worker binaries) subscribe and act on
// them. Today the only consumer persists notifications; push / email / SMS can
// be added as extra consumers on the same subject without touching producers.
package events

import (
	"context"
	"encoding/json"
	"coachwise/src/logger"
	"sync"
	"time"

	"coachwise/src/app/models"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// SubjectNotification carries NotificationJob messages.
const SubjectNotification = "events.notification"

// queueGroup load-balances a subject across every consumer instance, so running
// more worker processes just spreads the work (each message handled once).
const notificationQueueGroup = "notification-workers"

var (
	conn *nats.Conn
	mu   sync.RWMutex
)

// Connect opens the NATS connection. A blank url (or an unreachable server) is
// non-fatal: the bus stays disabled and Emit* become no-ops so the API keeps
// working without the queue.
func Connect(url string) {
	if url == "" {
		logger.Info("events: NATS url empty — event bus disabled")
		return
	}
	c, err := nats.Connect(url,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.Name("coachwise"),
	)
	if err != nil {
		logger.Errorf("events: NATS connect failed (%v) — event bus disabled", err)
		return
	}
	mu.Lock()
	conn = c
	mu.Unlock()
	logger.Infof("events: connected to NATS at %s", url)
}

func getConn() *nats.Conn {
	mu.RLock()
	defer mu.RUnlock()
	return conn
}

// NotificationJob is an event to notify a recipient. It is intentionally plain
// (no DB types) so producers, the wire format, and consumers stay decoupled.
type NotificationJob struct {
	UserID     uuid.UUID       `json:"user_id"`
	ActorID    *uuid.UUID      `json:"actor_id,omitempty"`
	Type       string          `json:"type"`
	EntityType *string         `json:"entity_type,omitempty"`
	EntityID   *uuid.UUID      `json:"entity_id,omitempty"`
	Data       json.RawMessage `json:"data,omitempty"`
}

// EmitNotification delivers a notification event. When the bus is actually
// connected it publishes to NATS (a consumer persists it + fans out to push /
// email / SMS). When the bus is down it FALLS BACK to writing the row directly,
// so in-app notifications always work. Never blocks/fails the caller's request.
//
// Note: with RetryOnFailedConnect a plain Publish "succeeds" (buffers) even while
// disconnected, so we gate on IsConnected() to detect a genuinely-up bus.
func EmitNotification(userID uuid.UUID, actorID *uuid.UUID, typ string, entityType *string, entityID *uuid.UUID, data any) {
	var raw json.RawMessage
	if data != nil {
		if b, err := json.Marshal(data); err == nil {
			raw = b
		}
	}

	if c := getConn(); c != nil && c.IsConnected() {
		job := NotificationJob{UserID: userID, ActorID: actorID, Type: typ, EntityType: entityType, EntityID: entityID, Data: raw}
		if payload, err := json.Marshal(job); err == nil {
			if err := c.Publish(SubjectNotification, payload); err == nil {
				EmitSignal(userID, "notifications") // poke the recipient's bell live
				return
			} else {
				logger.Errorf("events: publish failed, writing notification directly: %v", err)
			}
		}
	}

	// Fallback: bus unavailable → persist directly (push/email/SMS are skipped
	// until the queue is back, but the in-app notification is not lost).
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := models.InsertNotification(ctx, userID, actorID, typ, entityType, entityID, raw); err != nil {
		logger.Errorf("events: fallback insert notification: %v", err)
	}
}

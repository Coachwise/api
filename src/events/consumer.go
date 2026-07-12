package events

import (
	"context"
	"encoding/json"
	"coachwise/src/logger"
	"time"

	"coachwise/src/app/models"

	"github.com/nats-io/nats.go"
)

// StartNotificationConsumer subscribes to notification events and persists them.
// It uses a queue group, so this can run inside the API for dev and/or as many
// dedicated worker processes as needed in production — each message is handled
// once. No-op when the bus is disabled.
func StartNotificationConsumer() {
	c := getConn()
	if c == nil {
		logger.Info("events: notification consumer not started (bus disabled)")
		return
	}
	_, err := c.QueueSubscribe(SubjectNotification, notificationQueueGroup, func(m *nats.Msg) {
		var job NotificationJob
		if err := json.Unmarshal(m.Data, &job); err != nil {
			logger.Errorf("events: bad notification job: %v", err)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := models.InsertNotification(ctx, job.UserID, job.ActorID, job.Type, job.EntityType, job.EntityID, job.Data); err != nil {
			logger.Errorf("events: persist notification: %v", err)
		}
	})
	if err != nil {
		logger.Errorf("events: notification subscribe failed: %v", err)
		return
	}
	logger.Info("events: notification consumer subscribed")
}

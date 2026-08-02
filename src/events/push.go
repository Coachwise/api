package events

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"coachwise/src/app/models"
	"coachwise/src/logger"
	"coachwise/src/push"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// Its own queue group on the notification subject: NATS delivers each message
// once per group, so pushing runs alongside persisting rather than instead of it.
const pushQueueGroup = "push-workers"

// argActor resolves to the actor's display name; anything else is a key in the
// job's data blob.
const argActor = "@actor"

// Body args per notification type, in the order the device's format string
// expects. A type absent here gets a body with no args.
var pushBodyArgs = map[string][]string{
	models.NotifConnectionRequest:   {argActor},
	models.NotifConnectionAccepted:  {argActor},
	models.NotifMessageReceived:     {argActor},
	models.NotifPlanAssigned:        {argActor, "name"},
	models.NotifPlanRemoved:         {argActor, "name"},
	models.NotifPackageAssigned:     {argActor, "name"},
	models.NotifPackageRemoved:      {argActor, "name"},
	models.NotifPackageSubscribed:   {argActor, "name"},
	models.NotifAssessmentAssigned:  {argActor, "name"},
	models.NotifAssessmentSubmitted: {argActor, "name"},
	models.NotifBadgeGranted:        {"title"},
}

// StartPushConsumer sends a tray notification for every notification event.
func StartPushConsumer() {
	c := getConn()
	if c == nil {
		logger.Info("events: push consumer not started (bus disabled)")
		return
	}
	if !push.Enabled() {
		logger.Info("events: push consumer not started (push disabled)")
		return
	}
	_, err := c.QueueSubscribe(SubjectNotification, pushQueueGroup, func(m *nats.Msg) {
		var job NotificationJob
		if err := json.Unmarshal(m.Data, &job); err != nil {
			logger.Errorf("events: bad push job: %v", err)
			return
		}
		if err := deliverPush(job); err != nil {
			logger.Errorf("events: push job: %v", err)
			reportJobFailure("push", job.Type, err)
		}
	})
	if err != nil {
		logger.Errorf("events: push subscribe failed: %v", err)
		return
	}
	logger.Info("events: push consumer subscribed")
}

func deliverPush(job NotificationJob) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	devices, err := models.ListDeviceTokens(ctx, job.UserID)
	if err != nil {
		return err
	}
	if len(devices) == 0 {
		return nil
	}
	tokens := make([]string, 0, len(devices))
	for _, d := range devices {
		tokens = append(tokens, d.Token)
	}

	for _, t := range push.Deliver(ctx, tokens, buildMessage(job)) {
		if err := models.DeleteDeviceToken(ctx, t); err != nil {
			logger.Errorf("events: prune dead token: %v", err)
		}
	}
	return nil
}

// buildMessage turns a job into loc keys: PLAN_ASSIGNED → push_plan_assigned_
// {title,body}. The device owns every word.
func buildMessage(job NotificationJob) push.Message {
	base := "push_" + strings.ToLower(job.Type)
	m := push.Message{
		TitleLocKey: base + "_title",
		BodyLocKey:  base + "_body",
		BodyLocArgs: bodyArgs(job),
		Data:        map[string]string{"type": job.Type},
	}
	if job.EntityType != nil {
		m.Data["entity_type"] = *job.EntityType
	}
	if job.EntityID != nil {
		m.Data["entity_id"] = job.EntityID.String()
	}
	if job.ActorID != nil {
		m.Data["actor_id"] = job.ActorID.String()
	}
	return m
}

func bodyArgs(job NotificationJob) []string {
	spec := pushBodyArgs[job.Type]
	if len(spec) == 0 {
		return nil
	}
	var data map[string]any
	if len(job.Data) > 0 {
		_ = json.Unmarshal(job.Data, &data)
	}
	args := make([]string, 0, len(spec))
	for _, key := range spec {
		if key == argActor {
			args = append(args, actorName(job.ActorID))
			continue
		}
		if v, ok := data[key].(string); ok {
			args = append(args, v)
			continue
		}
		args = append(args, "")
	}
	return args
}

// actorName is the sender's display name. An arg is never dropped: the device's
// format string has a fixed placeholder count, and a short list would render the
// literal "%1$s".
func actorName(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	u, err := models.GetUser(*id)
	if err != nil {
		logger.Errorf("events: push actor lookup: %v", err)
		return ""
	}
	name := strings.TrimSpace(strings.Join([]string{deref(u.FirstName), deref(u.LastName)}, " "))
	if name == "" {
		name = u.Username
	}
	return name
}

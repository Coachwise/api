package models

import (
	"context"
	"time"

	"coachwise/src/database"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx/types"
)

// Notification types (kept in sync with the client's localized copy).
const (
	NotifConnectionRequest   = "CONNECTION_REQUEST"
	NotifConnectionAccepted  = "CONNECTION_ACCEPTED"
	NotifAssessmentAssigned  = "ASSESSMENT_ASSIGNED"
	NotifAssessmentSubmitted = "ASSESSMENT_SUBMITTED"
	NotifBadgeGranted        = "BADGE_GRANTED"
	NotifPackageSubscribed   = "PACKAGE_SUBSCRIBED" // client bought → notify coach
	NotifPackageAssigned     = "PACKAGE_ASSIGNED"   // coach enrolled a client → notify client
	NotifPackageRemoved      = "PACKAGE_REMOVED"    // coach removed a client → notify client
	NotifPlanAssigned        = "PLAN_ASSIGNED"      // coach assigned a plan → notify client
	NotifPlanRemoved         = "PLAN_REMOVED"       // coach unassigned a plan → notify client
	NotifSupportReply        = "SUPPORT_REPLY"      // support answered a ticket → notify the user
	NotifSupportUpdate       = "SUPPORT_UPDATE"     // a ticket's status changed (e.g. closed) → notify the user
	NotifMessageReceived     = "MESSAGE_RECEIVED"   // direct message → notify the recipient
)

// Notification is one in-app event for a recipient. The client renders localized
// copy from `Type` + `Data` (+ the hydrated actor), so no baked text is stored.
type Notification struct {
	ID         uuid.UUID  `db:"id" json:"id"`
	UserID     uuid.UUID  `db:"user_id" json:"user_id"`
	ActorID    *uuid.UUID `db:"actor_id" json:"actor_id,omitempty"`
	Type       string     `db:"type" json:"type"`
	EntityType *string    `db:"entity_type" json:"entity_type,omitempty"`
	EntityID   *uuid.UUID `db:"entity_id" json:"entity_id,omitempty"`
	Data       types.JSONText `db:"data" json:"data"`
	Read       bool       `db:"read" json:"read"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
	// Hydrated in fetch.sql (minimal user, no N+1).
	Actor     *User          `db:"-" json:"actor"`
	ActorJson types.JSONText `db:"actor" json:"-"`
}

func (Notification) TableName() string  { return "notifications" }
func (Notification) FetchQuery() string { return "notifications/fetch" }

// InsertNotification writes a notification row. Called by the queue consumer
// (producers publish an event; the consumer persists it here). `data` is raw
// JSON context (e.g. {"name": ...}); empty falls back to {}.
func InsertNotification(ctx context.Context, userID uuid.UUID, actorID *uuid.UUID, typ string, entityType *string, entityID *uuid.UUID, data []byte) error {
	if len(data) == 0 {
		data = []byte("{}")
	}
	// Chats would otherwise stack one row per message; collapse into the sender's
	// existing unread one.
	query := "notifications/create"
	if typ == NotifMessageReceived {
		query = "notifications/create_collapsed"
	}
	rows, err := database.Query(ctx, query, userID, actorID, typ, entityType, entityID, string(data))
	if err != nil {
		return err
	}
	rows.Close()
	return nil
}

// ListNotificationsPaginated returns a user's notifications, newest first.
func ListNotificationsPaginated(ctx context.Context, userID uuid.UUID, p database.Paginate) ([]Notification, int, error) {
	var (
		items     = []Notification{}
		fetchList []database.FetchList
		ids       []interface{}
		total     int
	)
	if err := database.QuerySelect("notifications/list", &fetchList, userID, p.Limit, p.Offset); err != nil {
		return nil, 0, err
	}
	if len(fetchList) < 1 {
		return items, 0, nil
	}
	total = fetchList[0].TotalCount
	for _, f := range fetchList {
		ids = append(ids, f.ID)
	}
	if err := database.Fetch(&items, ids...); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// CountUnreadNotifications returns how many unread notifications a user has.
func CountUnreadNotifications(ctx context.Context, userID uuid.UUID) (int, error) {
	var n int
	rows, err := database.Query(ctx, "notifications/unread_count", userID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if rows.Next() {
		if err := rows.Scan(&n); err != nil {
			return 0, err
		}
	}
	return n, nil
}

// MarkNotificationRead marks one notification read (owner only).
func MarkNotificationRead(ctx context.Context, id, userID uuid.UUID) error {
	rows, err := database.Query(ctx, "notifications/mark_read", id, userID)
	if err != nil {
		return err
	}
	rows.Close()
	return nil
}

// MarkAllNotificationsRead marks all of a user's notifications read.
func MarkAllNotificationsRead(ctx context.Context, userID uuid.UUID) error {
	rows, err := database.Query(ctx, "notifications/mark_all_read", userID)
	if err != nil {
		return err
	}
	rows.Close()
	return nil
}

package models

import (
	"context"
	"time"

	"coachwise/src/database"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx/types"
)

// AI message roles + statuses (mirror the ai_message_role / ai_message_status
// enums). Roles are only user | assistant; tool results live in Actions.
const (
	AIRoleUser      = "user"
	AIRoleAssistant = "assistant"

	AIStatusPending  = "pending"          // assistant turn queued, worker not done
	AIStatusAwaiting = "awaiting_approval" // has write proposals the user must confirm
	AIStatusDone     = "done"
	AIStatusFailed   = "failed"
)

// AIConversation is one chat thread with the assistant.
type AIConversation struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	UserID          uuid.UUID  `db:"user_id" json:"user_id"`
	Title           *string    `db:"title" json:"title"`
	Memory          string     `db:"memory" json:"-"`            // rolling summary, server-only
	SummarizedUntil *time.Time `db:"summarized_until" json:"-"`  // context cursor, server-only
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
	TotalCount      int        `db:"total_count" json:"-"` // set by list query only
}

// AIMessage is one turn. Actions holds proposed/executed actions (with results).
type AIMessage struct {
	ID               uuid.UUID      `db:"id" json:"id"`
	ConversationID   uuid.UUID      `db:"conversation_id" json:"conversation_id"`
	Role             string         `db:"role" json:"role"`
	Text             string         `db:"text" json:"text"`
	Actions          types.JSONText `db:"actions" json:"actions"`
	Status           string         `db:"status" json:"status"`
	Model            *string        `db:"model" json:"model,omitempty"`
	PromptTokens     int            `db:"prompt_tokens" json:"prompt_tokens"`
	CompletionTokens int            `db:"completion_tokens" json:"completion_tokens"`
	TotalTokens      int            `db:"total_tokens" json:"total_tokens"`
	CreatedAt        time.Time      `db:"created_at" json:"created_at"`
}

// CreateConversation opens a thread for a user.
func CreateConversation(ctx context.Context, userID uuid.UUID, title *string) (AIConversation, error) {
	var c AIConversation
	err := database.Get(&c, "ai_conversations/create", userID, title)
	return c, err
}

// GetConversation returns a conversation the user owns (else sql.ErrNoRows).
func GetConversation(ctx context.Context, id, userID uuid.UUID) (AIConversation, error) {
	var c AIConversation
	err := database.Get(&c, "ai_conversations/get", id, userID)
	return c, err
}

// ListConversations returns a user's threads, most-recent first.
func ListConversations(ctx context.Context, userID uuid.UUID, p database.Paginate) ([]AIConversation, int, error) {
	items := []AIConversation{}
	if err := database.QuerySelect("ai_conversations/list", &items, userID, p.Limit, p.Offset); err != nil {
		return nil, 0, err
	}
	total := 0
	if len(items) > 0 {
		total = items[0].TotalCount
	}
	return items, total, nil
}

// TouchConversation bumps updated_at (so a thread sorts to the top on activity).
func TouchConversation(ctx context.Context, id uuid.UUID) error {
	rows, err := database.Query(ctx, "ai_conversations/touch", id)
	if err != nil {
		return err
	}
	rows.Close()
	return nil
}

// UpdateConversationMemory stores the rolling summary + advances the cursor.
func UpdateConversationMemory(ctx context.Context, id uuid.UUID, memory string, until time.Time) error {
	rows, err := database.Query(ctx, "ai_conversations/update_memory", id, memory, until)
	if err != nil {
		return err
	}
	rows.Close()
	return nil
}

// InsertMessage writes one message and returns its id.
func InsertMessage(ctx context.Context, convID uuid.UUID, role, text, status string) (uuid.UUID, error) {
	var id uuid.UUID
	rows, err := database.Query(ctx, "ai_messages/create", convID, role, text, status)
	if err != nil {
		return uuid.Nil, err
	}
	defer rows.Close()
	if rows.Next() {
		if err := rows.Scan(&id); err != nil {
			return uuid.Nil, err
		}
	}
	return id, nil
}

// UpdateAssistantMessage fills a pending assistant turn with the model's output.
func UpdateAssistantMessage(ctx context.Context, id uuid.UUID, text string, actions []byte, status, model string, u AIUsage) error {
	if len(actions) == 0 {
		actions = []byte("[]")
	}
	rows, err := database.Query(ctx, "ai_messages/update_assistant",
		id, text, string(actions), status, model, u.PromptTokens, u.CompletionTokens, u.TotalTokens)
	if err != nil {
		return err
	}
	rows.Close()
	return nil
}

// SetMessageActions rewrites a message's actions (e.g. to record a write result)
// and its status.
func SetMessageActions(ctx context.Context, id uuid.UUID, actions []byte, status string) error {
	if len(actions) == 0 {
		actions = []byte("[]")
	}
	rows, err := database.Query(ctx, "ai_messages/set_actions", id, string(actions), status)
	if err != nil {
		return err
	}
	rows.Close()
	return nil
}

// GetAIMessage returns one message within a conversation.
func GetAIMessage(ctx context.Context, id, convID uuid.UUID) (AIMessage, error) {
	var m AIMessage
	err := database.Get(&m, "ai_messages/get", id, convID)
	return m, err
}

// FailAIMessage marks a pending assistant turn failed (no text — the client
// localizes from the status).
func FailAIMessage(ctx context.Context, id uuid.UUID) error {
	rows, err := database.Query(ctx, "ai_messages/fail", id)
	if err != nil {
		return err
	}
	rows.Close()
	return nil
}

// ListAIMessages returns a conversation's messages oldest-first (full thread).
func ListAIMessages(ctx context.Context, convID uuid.UUID) ([]AIMessage, error) {
	items := []AIMessage{}
	err := database.QuerySelect("ai_messages/list", &items, convID)
	return items, err
}

// WindowMessages returns messages after the summary cursor (the verbatim window
// fed to the model). A nil cursor returns the whole thread.
func WindowMessages(ctx context.Context, convID uuid.UUID, until *time.Time) ([]AIMessage, error) {
	items := []AIMessage{}
	err := database.QuerySelect("ai_messages/window", &items, convID, until)
	return items, err
}

// AIUsage is the token accounting persisted on an assistant turn.
type AIUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

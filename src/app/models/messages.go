package models

import (
	"context"
	"time"

	database "github.com/socious-io/pkg_database"

	"github.com/google/uuid"
)

type Message struct {
	ID        uuid.UUID  `db:"id" json:"id"`
	SenderID  uuid.UUID  `db:"sender_id" json:"sender_id"`
	ChatID    uuid.UUID  `db:"chat_id" json:"chat_id"`
	Body      string     `db:"body" json:"body"`
	MediaID   *uuid.UUID `db:"media_id" json:"media_id"`
	ReadAt    *time.Time `db:"read_at" json:"read_at"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	Media     *Media     `db:"-" json:"media,omitempty"`
}

type ThreadEntry struct {
	Message
}

func (Message) TableName() string {
	return "messages"
}

func (Message) FetchQuery() string {
	return "messages/fetch"
}

func CreateMessage(ctx context.Context, chatID uuid.UUID, senderID uuid.UUID, body string, mediaID *uuid.UUID) (*Message, error) {
	msg := &Message{
		SenderID: senderID,
		ChatID:   chatID,
		Body:     body,
		MediaID:  mediaID,
	}
	rows, err := database.Query(ctx, "messages/create", msg.ChatID, msg.SenderID, msg.Body, msg.MediaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(msg); err != nil {
			return nil, err
		}
	}

	return msg, nil
}

func ListMessages(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]Message, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	var msgs []Message
	if err := database.QuerySelect("messages/list_by_chat", &msgs, chatID, limit, offset); err != nil {
		return nil, err
	}
	return msgs, nil
}

func MarkChatRead(ctx context.Context, chatID uuid.UUID, userID uuid.UUID) error {
	_, err := database.Query(ctx, "messages/mark_chat_read", chatID, userID)
	return err
}

// ListThreads is a placeholder that returns an empty slice until thread aggregation is implemented.
func ListThreads(ctx context.Context, limit, offset int) ([]ThreadEntry, error) {
	return []ThreadEntry{}, nil
}

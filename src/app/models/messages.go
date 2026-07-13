package models

import (
	"context"
	"time"

	"coachwise/src/database"

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

// ThreadEntry is one conversation in the user's message list: the peer, the
// last message, and how many of the peer's messages are still unread.
type ThreadEntry struct {
	ChatID       uuid.UUID `json:"chat_id"`
	Peer         *User     `json:"peer"`
	LastMessage  string    `json:"last_message"`
	LastAt       time.Time `json:"last_at"`
	LastSenderID uuid.UUID `json:"last_sender_id"`
	UnreadCount  int       `json:"unread_count"`
}

type threadRow struct {
	ChatID       uuid.UUID `db:"chat_id"`
	PeerID       uuid.UUID `db:"peer_id"`
	LastBody     string    `db:"last_body"`
	LastAt       time.Time `db:"last_at"`
	LastSenderID uuid.UUID `db:"last_sender_id"`
	UnreadCount  int       `db:"unread_count"`
	TotalCount   int       `db:"total_count"`
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

// ListThreads returns the user's direct conversations (peer + last message +
// unread count), most recent first, plus the total count.
func ListThreads(ctx context.Context, userID uuid.UUID, p database.Paginate) ([]ThreadEntry, int, error) {
	var rows []threadRow
	if err := database.QuerySelect("messages/threads", &rows, userID, p.Limit, p.Offset); err != nil {
		return nil, 0, err
	}
	if len(rows) == 0 {
		return []ThreadEntry{}, 0, nil
	}
	total := rows[0].TotalCount

	ids := make([]interface{}, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.PeerID)
	}
	var peers []User
	if err := database.Fetch(&peers, ids...); err != nil {
		return nil, 0, err
	}
	byID := make(map[uuid.UUID]*User, len(peers))
	for i := range peers {
		byID[peers[i].ID] = &peers[i]
	}

	threads := make([]ThreadEntry, 0, len(rows))
	for _, r := range rows {
		threads = append(threads, ThreadEntry{
			ChatID:       r.ChatID,
			Peer:         byID[r.PeerID],
			LastMessage:  r.LastBody,
			LastAt:       r.LastAt,
			LastSenderID: r.LastSenderID,
			UnreadCount:  r.UnreadCount,
		})
	}
	return threads, total, nil
}

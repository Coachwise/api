package models

import (
	"context"
	"fmt"
	"sort"
	"time"

	database "github.com/socious-io/pkg_database"

	"github.com/google/uuid"
)

type Chat struct {
	ID        uuid.UUID `db:"id" json:"id"`
	Type      string    `db:"type" json:"type"`
	Name      *string   `db:"name" json:"name,omitempty"`
	OwnerID   uuid.UUID `db:"owner_id" json:"owner_id"`
	ChatKey   *string   `db:"chat_key" json:"chat_key,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type ChatMember struct {
	ChatID    uuid.UUID `db:"chat_id" json:"chat_id"`
	UserID    uuid.UUID `db:"user_id" json:"user_id"`
	Role      string    `db:"role" json:"role"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func directKey(a, b uuid.UUID) string {
	parts := []string{a.String(), b.String()}
	sort.Strings(parts)
	return parts[0] + ":" + parts[1]
}

func CreateChannel(ctx context.Context, ownerID uuid.UUID, name string) (*Chat, error) {
	var chat Chat
	rows, err := database.Query(ctx, "chats/create_channel", ownerID, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(&chat); err != nil {
			return nil, err
		}
	}
	if chat.ID == uuid.Nil {
		return nil, fmt.Errorf("failed to create channel")
	}
	// owner as OWNER
	_ = UpsertChatMember(ctx, chat.ID, ownerID, "OWNER")
	return &chat, nil
}

func GetOrCreateDirectChat(ctx context.Context, initiator uuid.UUID, other uuid.UUID) (*Chat, error) {
	key := directKey(initiator, other)
	var chat Chat
	if err := database.Get(&chat, "chats/fetch_by_key", key); err == nil {
		return &chat, nil
	}

	rows, err := database.Query(ctx, "chats/create_direct", initiator, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(&chat); err != nil {
			return nil, err
		}
	}
	if chat.ID == uuid.Nil {
		return nil, fmt.Errorf("failed to create direct chat")
	}

	// Add both members; initiator as OWNER, other as MEMBER
	_ = UpsertChatMember(ctx, chat.ID, initiator, "OWNER")
	_ = UpsertChatMember(ctx, chat.ID, other, "MEMBER")
	return &chat, nil
}

func UpsertChatMember(ctx context.Context, chatID, userID uuid.UUID, role string) error {
	_, err := database.Query(ctx, "chat_members/upsert", chatID, userID, role)
	return err
}

func CanSend(ctx context.Context, chatID, userID uuid.UUID) (bool, error) {
	// Permission enforcement not implemented; allow send for now
	return true, nil
}

func IsMember(ctx context.Context, chatID, userID uuid.UUID) (bool, error) {
	// Placeholder until chat membership queries are implemented
	return true, nil
}

func MemberRole(ctx context.Context, chatID, userID uuid.UUID) (string, error) {
	// Placeholder role to avoid hard failure in unimplemented chat membership
	return "OWNER", nil
}

func IsAdminOrOwner(ctx context.Context, chatID, userID uuid.UUID) (bool, error) {
	role, err := MemberRole(ctx, chatID, userID)
	if err != nil {
		return false, err
	}
	return role == "OWNER" || role == "ADMIN", nil
}

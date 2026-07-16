package models

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"coachwise/src/database"

	"github.com/jmoiron/sqlx"

	"github.com/google/uuid"
)

// scanOne reads exactly the first row of a RETURNING query into dest and closes
// the rows. Returns sql.ErrNoRows if the query produced nothing.
func scanOne(rows *sqlx.Rows, dest any) error {
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	return rows.StructScan(dest)
}

// Sender values on a support message.
const (
	SupportSenderUser   = "USER"
	SupportSenderAdmin  = "ADMIN"
	SupportSenderSystem = "SYSTEM"
)

// Ticket statuses and turns.
const (
	SupportStatusOpen   = "OPEN"
	SupportStatusClosed = "CLOSED"
	SupportTurnUser     = "USER"
	SupportTurnAdmin    = "ADMIN"
)

// Errors the send path returns so the view can map them to the right code.
var (
	ErrTicketNotFound = errors.New("ticket not found")
	ErrTicketClosed   = errors.New("ticket is closed")
	ErrNotYourTurn    = errors.New("waiting for the other side to reply")
)

type SupportTicket struct {
	ID            uuid.UUID `db:"id" json:"id"`
	UserID        uuid.UUID `db:"user_id" json:"user_id"`
	Subject       string    `db:"subject" json:"subject"`
	Status        string    `db:"status" json:"status"`
	Turn          string    `db:"turn" json:"turn"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
	LastMessageAt time.Time `db:"last_message_at" json:"last_message_at"`
}

type SupportMessage struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	TicketID    uuid.UUID  `db:"ticket_id" json:"ticket_id"`
	Sender      string     `db:"sender" json:"sender"`
	Body        string     `db:"body" json:"body"`
	DeliveredAt *time.Time `db:"delivered_at" json:"-"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
}

// SupportTicketListItem is a ticket plus a preview of its last message, for the
// user's ticket list.
type SupportTicketListItem struct {
	SupportTicket
	LastBody   *string `db:"last_body" json:"last_body"`
	LastSender *string `db:"last_sender" json:"last_sender"`
	TotalCount int     `db:"total_count" json:"-"`
}

// SupportDelivery is one admin/system message the worker has claimed to push.
type SupportDelivery struct {
	ID       uuid.UUID `db:"id"`
	TicketID uuid.UUID `db:"ticket_id"`
	UserID   uuid.UUID `db:"user_id"`
	Body     string    `db:"body"`
}

// OpenTicket creates a ticket and its first (USER) message atomically. The ticket
// opens awaiting the admin, which is the table default.
func OpenTicket(ctx context.Context, userID uuid.UUID, subject, body string) (*SupportTicket, *SupportMessage, error) {
	tx, err := database.GetDB().Beginx()
	if err != nil {
		return nil, nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	ticket := new(SupportTicket)
	rows, err := database.TxQuery(ctx, tx, "support/create_ticket", userID, subject)
	if err != nil {
		return nil, nil, err
	}
	if err := scanOne(rows, ticket); err != nil {
		return nil, nil, err
	}

	msg := new(SupportMessage)
	rows, err = database.TxQuery(ctx, tx, "support/create_message", ticket.ID, SupportSenderUser, body)
	if err != nil {
		return nil, nil, err
	}
	if err := scanOne(rows, msg); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	committed = true
	return ticket, msg, nil
}

// AddUserMessage appends a user reply, but only when it is the user's turn. The
// turn flip is a conditional UPDATE, so the guard and the flip are one atomic
// step; a prior fetch tells the caller which specific reason to report.
func AddUserMessage(ctx context.Context, ticketID, userID uuid.UUID, body string) (*SupportMessage, error) {
	ticket, err := GetTicket(ctx, ticketID, userID)
	if err != nil {
		return nil, err
	}
	if ticket == nil {
		return nil, ErrTicketNotFound
	}
	if ticket.Status == SupportStatusClosed {
		return nil, ErrTicketClosed
	}
	if ticket.Turn != SupportTurnUser {
		return nil, ErrNotYourTurn
	}

	tx, err := database.GetDB().Beginx()
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// The flip is the real guard: if another request already moved the turn, this
	// matches zero rows and we bail before inserting.
	flip, err := database.TxQuery(ctx, tx, "support/user_turn_flip", ticketID, userID)
	if err != nil {
		return nil, err
	}
	moved := flip.Next()
	flip.Close()
	if !moved {
		return nil, ErrNotYourTurn
	}

	msg := new(SupportMessage)
	rows, err := database.TxQuery(ctx, tx, "support/create_message", ticketID, SupportSenderUser, body)
	if err != nil {
		return nil, err
	}
	if err := scanOne(rows, msg); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	return msg, nil
}

// GetTicket returns the user's own ticket, or nil when it does not exist / is not
// theirs.
func GetTicket(ctx context.Context, ticketID, userID uuid.UUID) (*SupportTicket, error) {
	ticket := new(SupportTicket)
	if err := database.Get(ticket, "support/get_ticket", ticketID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return ticket, nil
}

// ListTicketMessages returns a ticket's messages oldest-first. The caller must
// have already checked ownership via GetTicket.
func ListTicketMessages(ctx context.Context, ticketID uuid.UUID) ([]SupportMessage, error) {
	msgs := []SupportMessage{}
	if err := database.QuerySelect("support/list_messages", &msgs, ticketID); err != nil {
		return nil, err
	}
	return msgs, nil
}

// ListUserTickets returns a user's tickets (most recently active first) plus the
// total for pagination.
func ListUserTickets(ctx context.Context, userID uuid.UUID, p database.Paginate) ([]SupportTicketListItem, int, error) {
	items := []SupportTicketListItem{}
	if err := database.QuerySelect("support/list_tickets", &items, userID, p.Limit, p.Offset); err != nil {
		return nil, 0, err
	}
	total := 0
	if len(items) > 0 {
		total = items[0].TotalCount
	}
	return items, total, nil
}

// ClaimUndeliveredReplies atomically claims every admin/system message not yet
// pushed to its user, returning what to deliver. Safe to run from many workers.
func ClaimUndeliveredReplies(ctx context.Context) ([]SupportDelivery, error) {
	out := []SupportDelivery{}
	rows, err := database.Query(ctx, "support/claim_undelivered")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var d SupportDelivery
		if err := rows.StructScan(&d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

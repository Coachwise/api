package models

import (
	"context"
	"errors"
	"time"

	database "github.com/socious-io/pkg_database"

	"github.com/google/uuid"
)

var (
	ErrInvalidAmount  = errors.New("invalid amount")
	ErrPayoutExceeds  = errors.New("payout exceeds available balance")
)

// Payout is a coach withdrawal (top-out) request. Draws from available balance.
type Payout struct {
	ID        uuid.UUID `db:"id" json:"id"`
	CoachID   uuid.UUID `db:"coach_id" json:"coach_id"`
	WalletID  uuid.UUID `db:"wallet_id" json:"wallet_id"`
	Amount    int64     `db:"amount" json:"amount"`
	Currency  string    `db:"currency" json:"currency"`
	Status    string    `db:"status" json:"status"`
	Note      *string   `db:"note" json:"note,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`

	TotalCount int `db:"total_count" json:"-"` // window count for pagination
}

// RequestPayout validates the amount against the coach's available balance, then
// records the request and debits the wallet — atomically.
func RequestPayout(ctx context.Context, coachID uuid.UUID, currency string, amount int64, note *string) (*Payout, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
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

	wallet, err := txGetOrCreateUserWallet(ctx, tx, coachID, currency)
	if err != nil {
		return nil, err
	}
	available, err := txWalletAvailable(ctx, tx, wallet.ID)
	if err != nil {
		return nil, err
	}
	if amount > available {
		return nil, ErrPayoutExceeds
	}

	payout := new(Payout)
	if err := txOne(ctx, tx, payout, "payouts/create", coachID, wallet.ID, amount, currency, note); err != nil {
		return nil, err
	}
	if err := txExec(ctx, tx, "wallet_transactions/create",
		wallet.ID, currency, -amount, "PAYOUT", time.Now().UTC(), "payout", payout.ID, "Payout request"); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	return payout, nil
}

// ListPayouts returns a coach's payout requests, newest first.
func ListPayoutsPaginated(coachID uuid.UUID, p database.Paginate) ([]Payout, int, error) {
	payouts := []Payout{}
	if err := database.QuerySelect("payouts/by_coach", &payouts, coachID, p.Limit, p.Offset); err != nil {
		return nil, 0, err
	}
	total := 0
	if len(payouts) > 0 {
		total = payouts[0].TotalCount
	}
	return payouts, total, nil
}

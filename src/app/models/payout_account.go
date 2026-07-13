package models

import (
	"errors"
	"time"

	"coachwise/src/database"

	"github.com/google/uuid"
)

// ErrPayoutAccountNotFound means the coach hasn't set a payout destination for
// the requested currency yet.
var ErrPayoutAccountNotFound = errors.New("payout account not set up")

// PayoutAccount is a coach's payout destination for one currency. The method is
// currency-driven: IRR uses CARD; other currencies will use STRIPE (or a BANK
// account-to-account transfer) once wired. Only the coach reads their own row.
type PayoutAccount struct {
	ID              uuid.UUID `db:"id" json:"id"`
	UserID          uuid.UUID `db:"user_id" json:"user_id"`
	Currency        string    `db:"currency" json:"currency"`
	Method          string    `db:"method" json:"method"`
	AccountHolder   *string   `db:"account_holder" json:"account_holder,omitempty"`
	CardNumber      *string   `db:"card_number" json:"card_number,omitempty"`
	IBAN            *string   `db:"iban" json:"iban,omitempty"`
	BankName        *string   `db:"bank_name" json:"bank_name,omitempty"`
	Swift           *string   `db:"swift" json:"swift,omitempty"`
	StripeAccountID *string   `db:"stripe_account_id" json:"stripe_account_id,omitempty"`
	Status          string    `db:"status" json:"status"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

// GetPayoutAccount returns the coach's payout account for a currency, or
// ErrPayoutAccountNotFound if none is set up.
func GetPayoutAccount(userID uuid.UUID, currency string) (*PayoutAccount, error) {
	a := new(PayoutAccount)
	if err := database.Get(a, "payout_accounts/get", userID, currency); err != nil {
		return nil, ErrPayoutAccountNotFound
	}
	return a, nil
}

// UpsertPayoutAccount creates or replaces the coach's payout account for its
// currency. Editing details resets verification to UNVERIFIED.
func UpsertPayoutAccount(a *PayoutAccount) (*PayoutAccount, error) {
	out := new(PayoutAccount)
	if err := database.Get(out, "payout_accounts/upsert",
		a.UserID, a.Currency, a.Method, a.AccountHolder, a.CardNumber,
		a.IBAN, a.BankName, a.Swift, a.StripeAccountID, "UNVERIFIED",
	); err != nil {
		return nil, err
	}
	return out, nil
}

package models

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"coachwise/src/database"

	"coachwise/src/config"
	"coachwise/src/payments"

	"github.com/google/uuid"
)

var (
	ErrPaymentNotFound     = errors.New("payment not found")
	ErrTopUpAmountMismatch = errors.New("verified amount does not match the request")
)

// GetPayment fetches a top-up / money-in record by id.
func GetPayment(id uuid.UUID) (*Payment, error) {
	p := new(Payment)
	if err := database.Fetch(p, id); err != nil {
		return nil, ErrPaymentNotFound
	}
	return p, nil
}

// InitiateRedirectTopUp records a PENDING top-up and opens a gateway session,
// returning the URL to send the buyer's browser to. The gateway posts its result
// back to config CallbackURL; SettleRedirectTopUp finishes the job on callback.
func InitiateRedirectTopUp(ctx context.Context, userID uuid.UUID, amount int64, provider, cellNumber string) (*Payment, string, error) {
	if amount <= 0 {
		return nil, "", ErrInvalidAmount
	}
	currency, err := ValidateCurrency(DefaultCurrency())
	if err != nil {
		return nil, "", err
	}
	wallet, err := GetOrCreateUserWallet(userID, currency)
	if err != nil {
		return nil, "", err
	}

	var noOrder *uuid.UUID
	var noRef *string
	pay := new(Payment)
	// Store the registered provider name so the callback can verify with the same
	// gateway.
	rows, err := database.Query(ctx, "payments/create", noOrder, userID, wallet.ID, amount, currency, provider, noRef, "PENDING")
	if err != nil {
		return nil, "", err
	}
	for rows.Next() {
		if err := rows.StructScan(pay); err != nil {
			rows.Close()
			return nil, "", err
		}
	}
	rows.Close()

	// ResNum = our payment id, so the callback can find this record.
	redirectURL, _, err := payments.GetToken(provider, amount, currency, pay.ID.String(), config.Config.Payments.CallbackURL, cellNumber)
	if err != nil {
		_ = failPayment(pay.ID)
		return nil, "", err
	}
	return pay, redirectURL, nil
}

// SettleRedirectTopUp finishes a redirect top-up from the gateway callback: it
// verifies the transaction, checks the amount matches, and credits the wallet
// exactly once (idempotent across duplicate callbacks). Returns whether the
// wallet was credited.
func SettleRedirectTopUp(ctx context.Context, paymentID uuid.UUID, refNum string, stateOK bool) (*Payment, bool, error) {
	pay, err := GetPayment(paymentID)
	if err != nil {
		return nil, false, err
	}
	if pay.Status != "PENDING" {
		return pay, false, nil // already settled — idempotent
	}
	if !stateOK {
		_ = failPayment(paymentID)
		return pay, false, nil
	}
	amount, ok, err := payments.Verify(pay.Provider, refNum)
	if err != nil {
		return pay, false, err // transient gateway error — let the caller retry, don't fail yet
	}
	if !ok {
		_ = failPayment(paymentID)
		return pay, false, nil
	}
	if amount != pay.Amount {
		_ = failPayment(paymentID)
		return pay, false, ErrTopUpAmountMismatch
	}

	tx, err := database.GetDB().Beginx()
	if err != nil {
		return pay, false, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// The PENDING guard in the UPDATE makes crediting happen exactly once even if
	// two callbacks race — the second gets no row back.
	settled := new(Payment)
	if err := txOne(ctx, tx, settled, "payments/settle", paymentID, "PAID", refNum); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return pay, false, nil // a concurrent callback already settled it
		}
		return pay, false, err
	}
	if err := txExec(ctx, tx, "wallet_transactions/create",
		settled.WalletID, settled.Currency, settled.Amount, "TOPUP", time.Now().UTC(), "payment", settled.ID, "Top-up"); err != nil {
		return pay, false, err
	}
	if err := tx.Commit(); err != nil {
		return pay, false, err
	}
	committed = true
	return settled, true, nil
}

func failPayment(id uuid.UUID) error {
	rows, err := database.Query(context.Background(), "payments/settle", id, "FAILED", "")
	if err != nil {
		return err
	}
	rows.Close()
	return nil
}

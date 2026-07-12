package models

import (
	"context"
	"database/sql"
	"time"

	database "github.com/socious-io/pkg_database"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Wallet holds funds for one owner in one currency. A NULL owner is the platform
// system wallet (fees + Pro revenue).
type Wallet struct {
	ID        uuid.UUID  `db:"id" json:"id"`
	OwnerID   *uuid.UUID `db:"owner_id" json:"owner_id"`
	Currency  string     `db:"currency" json:"currency"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
}

// WalletBalance is derived from the ledger: available = settled credits − debits;
// pending = credits still in escrow (available_at in the future).
type WalletBalance struct {
	Available int64  `db:"available" json:"available"`
	Pending   int64  `db:"pending" json:"pending"`
	Currency  string `db:"-" json:"currency"`
}

// WalletIncome is cumulative earnings (SALE credits): all-time total + the
// current calendar month. Distinct from the spendable balance (which nets out
// payouts and escrow).
type WalletIncome struct {
	Total    int64  `db:"total" json:"total"`
	Month    int64  `db:"month" json:"month"`
	Currency string `db:"-" json:"currency"`
}

// WalletTransaction is one signed ledger entry.
type WalletTransaction struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	WalletID    uuid.UUID  `db:"wallet_id" json:"wallet_id"`
	Currency    string     `db:"currency" json:"currency"`
	Amount      int64      `db:"amount" json:"amount"`
	Type        string     `db:"type" json:"type"`
	AvailableAt time.Time  `db:"available_at" json:"available_at"`
	RefType     *string    `db:"ref_type" json:"ref_type,omitempty"`
	RefID       *uuid.UUID `db:"ref_id" json:"ref_id,omitempty"`
	Description *string    `db:"description" json:"description,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
}

// ---- Non-transactional helpers (wallet endpoints) ----

// GetOrCreateUserWallet returns the user's wallet for a currency, creating it if
// it doesn't exist yet.
func GetOrCreateUserWallet(ownerID uuid.UUID, currency string) (*Wallet, error) {
	w := new(Wallet)
	if err := database.Get(w, "wallets/by_owner", ownerID, currency); err == nil {
		return w, nil
	}
	rows, err := database.Query(context.Background(), "wallets/create", ownerID, currency)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		if err := rows.StructScan(w); err != nil {
			return nil, err
		}
	}
	return w, nil
}

// CreditTopUp records a completed manual top-up: a PAID payment row plus a TOPUP
// ledger credit. The provider charge already succeeded before this is called.
func CreditTopUp(ctx context.Context, userID uuid.UUID, wallet *Wallet, amount int64, providerRef string) error {
	tx, err := database.GetDB().Beginx()
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var noOrder *uuid.UUID
	pay := new(Payment)
	if err := txOne(ctx, tx, pay, "payments/create",
		noOrder, userID, wallet.ID, amount, wallet.Currency, "STUB", providerRef, "PAID"); err != nil {
		return err
	}
	if err := txExec(ctx, tx, "wallet_transactions/create",
		wallet.ID, wallet.Currency, amount, "TOPUP", time.Now().UTC(), "payment", pay.ID, "Top-up"); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

// WalletBalanceOf returns the derived available/pending balance for a wallet.
func WalletBalanceOf(walletID uuid.UUID, currency string) (*WalletBalance, error) {
	b := new(WalletBalance)
	if err := database.Get(b, "wallets/balance", walletID); err != nil {
		return nil, err
	}
	b.Currency = currency
	return b, nil
}

// WalletIncomeOf aggregates SALE credits (all-time total + current month).
func WalletIncomeOf(walletID uuid.UUID, currency string) (*WalletIncome, error) {
	inc := new(WalletIncome)
	if err := database.Get(inc, "wallets/income", walletID); err != nil {
		return nil, err
	}
	inc.Currency = currency
	return inc, nil
}

// ListWalletTransactions returns a wallet's ledger, newest first.
func ListWalletTransactions(walletID uuid.UUID, limit, offset int) ([]WalletTransaction, error) {
	txns := []WalletTransaction{}
	if err := database.QuerySelect("wallet_transactions/list", &txns, walletID, limit, offset); err != nil {
		return nil, err
	}
	return txns, nil
}

// ---- Transactional helpers (purchase flow) ----

func txOne(ctx context.Context, tx *sqlx.Tx, dest interface{}, name string, args ...interface{}) error {
	rows, err := database.TxQuery(ctx, tx, name, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return sql.ErrNoRows
	}
	return rows.StructScan(dest)
}

func txExec(ctx context.Context, tx *sqlx.Tx, name string, args ...interface{}) error {
	rows, err := database.TxQuery(ctx, tx, name, args...)
	if err != nil {
		return err
	}
	rows.Close()
	return nil
}

// addEntry writes one ledger entry inside the tx.
func addEntry(ctx context.Context, tx *sqlx.Tx, walletID uuid.UUID, currency string, amount int64, kind string, availableAt time.Time, refID uuid.UUID, desc string) error {
	refType := "order"
	return txExec(ctx, tx, "wallet_transactions/create", walletID, currency, amount, kind, availableAt, refType, refID, desc)
}

func txGetOrCreateUserWallet(ctx context.Context, tx *sqlx.Tx, ownerID uuid.UUID, currency string) (*Wallet, error) {
	w := new(Wallet)
	if err := txOne(ctx, tx, w, "wallets/by_owner", ownerID, currency); err == nil {
		return w, nil
	}
	if err := txOne(ctx, tx, w, "wallets/create", ownerID, currency); err != nil {
		return nil, err
	}
	return w, nil
}

func txGetOrCreatePlatformWallet(ctx context.Context, tx *sqlx.Tx, currency string) (*Wallet, error) {
	w := new(Wallet)
	if err := txOne(ctx, tx, w, "wallets/platform", currency); err == nil {
		return w, nil
	}
	if err := txOne(ctx, tx, w, "wallets/create_platform", currency); err != nil {
		return nil, err
	}
	return w, nil
}

func txWalletAvailable(ctx context.Context, tx *sqlx.Tx, walletID uuid.UUID) (int64, error) {
	b := new(WalletBalance)
	if err := txOne(ctx, tx, b, "wallets/balance", walletID); err != nil {
		return 0, err
	}
	return b.Available, nil
}

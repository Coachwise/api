package models

import (
	"context"
	"errors"
	"time"

	"coachwise/src/database"

	"coachwise/src/logger"
	"coachwise/src/payments"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	ErrPaymentFailed   = errors.New("payment failed")
	ErrPackageInactive = errors.New("package is not available")
	ErrSelfPurchase    = errors.New("cannot purchase your own package")
)

// Order is the audit record for a purchase (Pro or a coach package).
type Order struct {
	ID             uuid.UUID  `db:"id" json:"id"`
	BuyerID        uuid.UUID  `db:"buyer_id" json:"buyer_id"`
	Kind           string     `db:"kind" json:"kind"`
	Currency       string     `db:"currency" json:"currency"`
	CoachID        *uuid.UUID `db:"coach_id" json:"coach_id,omitempty"`
	PackageID      *uuid.UUID `db:"package_id" json:"package_id,omitempty"`
	DurationMonths int        `db:"duration_months" json:"duration_months"`
	UnitAmount     int64      `db:"unit_amount" json:"unit_amount"`
	Subtotal       int64      `db:"subtotal" json:"subtotal"`
	DiscountAmount int64      `db:"discount_amount" json:"discount_amount"`
	FeeAmount      int64      `db:"fee_amount" json:"fee_amount"`
	Total          int64      `db:"total" json:"total"`
	Status         string     `db:"status" json:"status"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at" json:"updated_at"`
}

// Payment is a money-in event (top-up).
type Payment struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	OrderID     *uuid.UUID `db:"order_id" json:"order_id,omitempty"`
	UserID      uuid.UUID  `db:"user_id" json:"user_id"`
	WalletID    uuid.UUID  `db:"wallet_id" json:"wallet_id"`
	Amount      int64      `db:"amount" json:"amount"`
	Currency    string     `db:"currency" json:"currency"`
	Provider    string     `db:"provider" json:"provider"`
	ProviderRef *string    `db:"provider_ref" json:"provider_ref,omitempty"`
	Status      string     `db:"status" json:"status"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

func (Payment) TableName() string  { return "payments" }
func (Payment) FetchQuery() string { return "payments/fetch" }

// autoTopUp tops the buyer's wallet up by exactly the shortfall (if any) via the
// chosen payment provider, so the buy flow feels like a single pay-per-purchase
// step. The provider must handle the wallet's currency.
func autoTopUp(ctx context.Context, tx *sqlx.Tx, provider string, buyerID uuid.UUID, wallet *Wallet, total int64) error {
	available, err := txWalletAvailable(ctx, tx, wallet.ID)
	if err != nil {
		return err
	}
	shortfall := total - available
	if shortfall <= 0 {
		return nil
	}
	res, err := payments.Charge(provider, buyerID, shortfall, wallet.Currency)
	if err == payments.ErrNoProvider || err == payments.ErrCurrencyUnsupported {
		return err // surfaced distinctly to the buyer
	}
	if err != nil || res.Status != "PAID" {
		return ErrPaymentFailed
	}
	var noOrder *uuid.UUID
	pay := new(Payment)
	if err := txOne(ctx, tx, pay, "payments/create",
		noOrder, buyerID, wallet.ID, shortfall, wallet.Currency, provider, res.Ref, "PAID"); err != nil {
		return err
	}
	return txExec(ctx, tx, "wallet_transactions/create",
		wallet.ID, wallet.Currency, shortfall, "TOPUP", time.Now().UTC(), "payment", pay.ID, "Top-up")
}

// PurchasePro charges the buyer for `months` of Pro in a currency (via the chosen
// provider) and extends pro_until. Pro is 100% platform revenue.
func PurchasePro(ctx context.Context, buyerID uuid.UUID, currency, provider string, months int) (*Order, error) {
	currency, err := ValidateCurrency(currency)
	if err != nil {
		return nil, err
	}
	quote, err := QuotePro(currency, months)
	if err != nil {
		return nil, err
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

	buyerWallet, err := txGetOrCreateUserWallet(ctx, tx, buyerID, currency)
	if err != nil {
		return nil, err
	}
	if err := autoTopUp(ctx, tx, provider, buyerID, buyerWallet, quote.Total); err != nil {
		return nil, err
	}

	var noUUID *uuid.UUID
	order := new(Order)
	if err := txOne(ctx, tx, order, "orders/create",
		buyerID, "PRO", currency, noUUID, noUUID, months,
		quote.UnitAmount, quote.Subtotal, quote.DiscountAmount, quote.FeeAmount, quote.Total, "PAID"); err != nil {
		return nil, err
	}
	if err := addEntry(ctx, tx, buyerWallet.ID, currency, -quote.Total, "PURCHASE", time.Now().UTC(), order.ID, "Pro membership"); err != nil {
		return nil, err
	}
	platform, err := txGetOrCreatePlatformWallet(ctx, tx, currency)
	if err != nil {
		return nil, err
	}
	if err := addEntry(ctx, tx, platform.ID, currency, quote.Total, "SALE", time.Now().UTC(), order.ID, "Pro revenue"); err != nil {
		return nil, err
	}
	if err := txExec(ctx, tx, "users/extend_pro", buyerID, months); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	return order, nil
}

// PurchasePackage charges the buyer for `months` of a coach package: credits the
// coach's wallet (net of the platform fee, held in escrow), credits the platform
// fee, enrolls the client (+ assigns plans) and grants Pro for the duration.
func PurchasePackage(ctx context.Context, buyerID, packageID uuid.UUID, currency, provider string, months int, buyerIsPro bool) (*Order, error) {
	pkg, err := GetPackage(ctx, packageID)
	if err != nil {
		return nil, err
	}
	if !pkg.IsActive {
		return nil, ErrPackageInactive
	}
	if pkg.CoachID == buyerID {
		return nil, ErrSelfPurchase
	}
	currency, err = ValidateCurrency(currency)
	if err != nil {
		return nil, err
	}
	quote, err := QuotePackage(pkg, currency, months, buyerIsPro)
	if err != nil {
		return nil, err
	}
	settings, err := GetPlatformSettings()
	if err != nil {
		return nil, err
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

	buyerWallet, err := txGetOrCreateUserWallet(ctx, tx, buyerID, currency)
	if err != nil {
		return nil, err
	}
	if err := autoTopUp(ctx, tx, provider, buyerID, buyerWallet, quote.Total); err != nil {
		return nil, err
	}

	order := new(Order)
	if err := txOne(ctx, tx, order, "orders/create",
		buyerID, "PACKAGE", currency, pkg.CoachID, pkg.ID, quote.ProMonths,
		quote.UnitAmount, quote.Subtotal, quote.DiscountAmount, quote.FeeAmount, quote.Total, "PAID"); err != nil {
		return nil, err
	}
	if err := addEntry(ctx, tx, buyerWallet.ID, currency, -quote.Total, "PURCHASE", time.Now().UTC(), order.ID, "Package: "+pkg.Name); err != nil {
		return nil, err
	}
	// Coach earnings held in escrow until available_at (now + escrow_hold_days).
	coachWallet, err := txGetOrCreateUserWallet(ctx, tx, pkg.CoachID, currency)
	if err != nil {
		return nil, err
	}
	availableAt := time.Now().UTC().AddDate(0, 0, settings.EscrowHoldDays)
	if err := addEntry(ctx, tx, coachWallet.ID, currency, quote.CoachNet, "SALE", availableAt, order.ID, "Package sale: "+pkg.Name); err != nil {
		return nil, err
	}
	// Platform income: the fee on the package + (when the buyer wasn't Pro) the
	// bundled Pro price. Pro revenue is available immediately (not escrowed).
	if quote.FeeAmount > 0 || quote.ProAmount > 0 {
		platform, err := txGetOrCreatePlatformWallet(ctx, tx, currency)
		if err != nil {
			return nil, err
		}
		if quote.FeeAmount > 0 {
			if err := addEntry(ctx, tx, platform.ID, currency, quote.FeeAmount, "FEE", time.Now().UTC(), order.ID, "Platform fee"); err != nil {
				return nil, err
			}
		}
		if quote.ProAmount > 0 {
			if err := addEntry(ctx, tx, platform.ID, currency, quote.ProAmount, "SALE", time.Now().UTC(), order.ID, "Pro membership"); err != nil {
				return nil, err
			}
		}
	}
	// Grant Pro only when the buyer paid for it here (they weren't already Pro).
	// An already-Pro buyer keeps their existing membership.
	if quote.ProIncluded && quote.ProMonths > 0 {
		if err := txExec(ctx, tx, "users/extend_pro", buyerID, quote.ProMonths); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true

	// Post-commit: enroll the client (+ assign the package's plans). A repeat
	// purchase (already enrolled) is a renewal — not an error.
	if _, err := EnrollClient(ctx, packageID, pkg.CoachID, buyerID); err != nil && err != ErrClientHasPackage {
		logger.Errorf("purchase %s: enroll client failed: %v", order.ID, err)
	}
	return order, nil
}

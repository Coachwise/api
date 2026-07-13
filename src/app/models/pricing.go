package models

import (
	"errors"

	"coachwise/src/database"
)

// ErrPriceNotConfigured means no price row exists for the requested currency.
var ErrPriceNotConfigured = errors.New("price not configured")

// ErrInvalidDuration means the requested months isn't an offered duration tier.
var ErrInvalidDuration = errors.New("unsupported duration")

// PlatformSettings holds the DB-backed knobs an admin dashboard will edit later.
type PlatformSettings struct {
	CoachFeePercent  int    `db:"coach_fee_percent" json:"coach_fee_percent"`
	EscrowHoldDays   int    `db:"escrow_hold_days" json:"escrow_hold_days"`
	DefaultCurrency  string `db:"default_currency" json:"default_currency"`
	OneTimeProMonths int    `db:"one_time_pro_months" json:"one_time_pro_months"`
}

// DurationTier is a purchasable duration and its discount (shared by Pro + packages).
type DurationTier struct {
	Months          int `db:"months" json:"months"`
	DiscountPercent int `db:"discount_percent" json:"discount_percent"`
}

// Quote is the fully-computed price breakdown for a purchase. All amounts are the
// currency's whole unit (Toman for IRR). Money math lives here, never on the client.
type Quote struct {
	Kind            string `json:"kind"`     // PRO | PACKAGE
	Currency        string `json:"currency"`
	Months          int    `json:"months"`
	UnitAmount      int64  `json:"unit_amount"`      // monthly price
	Subtotal        int64  `json:"subtotal"`         // unit * months
	DiscountPercent int    `json:"discount_percent"` // from the duration tier
	DiscountAmount  int64  `json:"discount_amount"`
	Total           int64  `json:"total"`     // what the buyer pays (package + Pro add-on)
	FeePercent      int    `json:"fee_percent"`
	FeeAmount       int64  `json:"fee_amount"` // platform fee on the package portion (PACKAGE only)
	CoachNet        int64  `json:"coach_net"`  // package-after-discount − fee (PACKAGE only)
	OneTime         bool   `json:"one_time"`   // ONE_TIME package (flat, lifetime)
	ProMonths       int    `json:"pro_months"` // months of Pro this purchase grants
	// A coach package requires platform Pro. When the buyer isn't already Pro, the
	// Pro price for pro_months is added to the total (goes 100% to the platform).
	ProIncluded bool  `json:"pro_included"`
	ProAmount   int64 `json:"pro_amount"`
}

func GetPlatformSettings() (*PlatformSettings, error) {
	s := new(PlatformSettings)
	if err := database.Get(s, "pricing/settings"); err != nil {
		return nil, err
	}
	return s, nil
}

func ListDurationTiers() ([]DurationTier, error) {
	tiers := []DurationTier{}
	if err := database.QuerySelect("pricing/tiers", &tiers); err != nil {
		return nil, err
	}
	return tiers, nil
}

// durationDiscount returns the discount percent for months, or ErrInvalidDuration.
func durationDiscount(months int) (int, error) {
	t := new(DurationTier)
	if err := database.Get(t, "pricing/tier", months); err != nil {
		return 0, ErrInvalidDuration
	}
	return t.DiscountPercent, nil
}

func proMonthlyPrice(currency string) (int64, error) {
	var row struct {
		MonthlyAmount int64 `db:"monthly_amount"`
	}
	if err := database.Get(&row, "pricing/pro_price", currency); err != nil {
		return 0, ErrPriceNotConfigured
	}
	return row.MonthlyAmount, nil
}

// compute fills in a Quote's derived amounts from unit/months/discount/fee.
func compute(kind, currency string, unit int64, months, discountPct, feePct int) *Quote {
	subtotal := unit * int64(months)
	discount := subtotal * int64(discountPct) / 100
	total := subtotal - discount
	fee := total * int64(feePct) / 100
	return &Quote{
		Kind:            kind,
		Currency:        currency,
		Months:          months,
		UnitAmount:      unit,
		Subtotal:        subtotal,
		DiscountPercent: discountPct,
		DiscountAmount:  discount,
		Total:           total,
		FeePercent:      feePct,
		FeeAmount:       fee,
		CoachNet:        total - fee,
	}
}

// QuotePro prices a Pro membership for the given currency and duration. Pro is
// 100% platform revenue (no coach, no fee).
func QuotePro(currency string, months int) (*Quote, error) {
	unit, err := proMonthlyPrice(currency)
	if err != nil {
		return nil, err
	}
	discount, err := durationDiscount(months)
	if err != nil {
		return nil, err
	}
	q := compute("PRO", currency, unit, months, discount, 0)
	q.ProMonths = months
	return q, nil
}

// QuotePackage prices a coach package in a currency. SUBSCRIPTION packages use
// the chosen months with the shared duration tiers; ONE_TIME packages are a flat
// single charge (months ignored). The platform fee % is applied to the package
// portion; the remainder is the coach's net (escrow credit). ProMonths is how
// many months of Pro the purchase grants. When buyerIsPro is false, the Pro price
// for ProMonths is added to the total (Pro is required to have a coach).
func QuotePackage(pkg *CoachPackage, currency string, months int, buyerIsPro bool) (*Quote, error) {
	unit, err := GetPackagePrice(pkg.ID, currency)
	if err != nil {
		return nil, err
	}
	settings, err := GetPlatformSettings()
	if err != nil {
		return nil, err
	}

	var q *Quote
	if pkg.BillingType == "ONE_TIME" {
		q = compute("PACKAGE", currency, unit, 1, 0, settings.CoachFeePercent)
		q.OneTime = true
		q.ProMonths = settings.OneTimeProMonths
	} else {
		discount, derr := durationDiscount(months)
		if derr != nil {
			return nil, derr
		}
		q = compute("PACKAGE", currency, unit, months, discount, settings.CoachFeePercent)
		q.ProMonths = months
	}

	// Bundle Pro when the buyer doesn't already have it.
	if !buyerIsPro && q.ProMonths > 0 {
		proQ, perr := QuotePro(currency, q.ProMonths)
		if perr != nil {
			return nil, perr
		}
		q.ProIncluded = true
		q.ProAmount = proQ.Total
		q.Total += proQ.Total
	}
	return q, nil
}

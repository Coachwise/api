package models

import (
	"context"

	database "github.com/socious-io/pkg_database"

	"coachwise/src/payments"

	"github.com/google/uuid"
)

// PackagePrice is a coach package's price in one currency. `amount` is the
// currency's whole unit — the monthly price for SUBSCRIPTION packages, the flat
// price for ONE_TIME packages.
type PackagePrice struct {
	Currency string `db:"currency" json:"currency"`
	Amount   int64  `db:"amount" json:"amount"`
}

// PurchaseOption is a currency a package can be bought in, plus the providers
// that handle that currency. Drives the buyer's currency + provider pickers.
type PurchaseOption struct {
	Currency  string                 `json:"currency"`
	Amount    int64                  `json:"amount"`
	Providers []payments.Descriptor  `json:"providers"`
}

// ListPackagePrices returns all per-currency prices for a package.
func ListPackagePrices(packageID uuid.UUID) ([]PackagePrice, error) {
	prices := []PackagePrice{}
	if err := database.QuerySelect("package_prices/list", &prices, packageID); err != nil {
		return nil, err
	}
	return prices, nil
}

// GetPackagePrice returns a package's price in a currency, or ErrPriceNotConfigured.
func GetPackagePrice(packageID uuid.UUID, currency string) (int64, error) {
	p := new(PackagePrice)
	if err := database.Get(p, "package_prices/get", packageID, currency); err != nil {
		return 0, ErrPriceNotConfigured
	}
	return p.Amount, nil
}

// SetPackagePrice upserts a package's price in a currency.
func SetPackagePrice(ctx context.Context, packageID uuid.UUID, currency string, amount int64) error {
	rows, err := database.Query(ctx, "package_prices/upsert", packageID, currency, amount)
	if err != nil {
		return err
	}
	rows.Close()
	return nil
}

// DeletePackagePrice removes a package's price in a currency.
func DeletePackagePrice(ctx context.Context, packageID uuid.UUID, currency string) error {
	rows, err := database.Query(ctx, "package_prices/delete", packageID, currency)
	if err != nil {
		return err
	}
	rows.Close()
	return nil
}

// PackagePurchaseOptions returns the currencies a package can be bought in that
// also have at least one payment provider — each with its providers.
func PackagePurchaseOptions(packageID uuid.UUID) ([]PurchaseOption, error) {
	prices, err := ListPackagePrices(packageID)
	if err != nil {
		return nil, err
	}
	opts := []PurchaseOption{}
	for _, p := range prices {
		providers := payments.ProvidersFor(p.Currency)
		if len(providers) == 0 {
			continue // no way to pay in this currency yet — hide it
		}
		opts = append(opts, PurchaseOption{Currency: p.Currency, Amount: p.Amount, Providers: providers})
	}
	return opts, nil
}

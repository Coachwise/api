package models

import (
	"errors"
	"strings"

	database "github.com/socious-io/pkg_database"
)

// ErrUnsupportedCurrency means the code isn't a platform-supported (enabled)
// currency.
var ErrUnsupportedCurrency = errors.New("unsupported currency")

// Currency is a platform-supported currency. `decimals` is the minor-unit digit
// count (0 for Toman). Amounts everywhere are stored in the whole unit.
type Currency struct {
	Code     string `db:"code" json:"code"`
	Name     string `db:"name" json:"name"`
	Symbol   string `db:"symbol" json:"symbol"`
	Decimals int    `db:"decimals" json:"decimals"`
	Enabled  bool   `db:"enabled" json:"enabled"`
}

// ListCurrencies returns the enabled currencies.
func ListCurrencies() ([]Currency, error) {
	currencies := []Currency{}
	if err := database.QuerySelect("currencies/list", &currencies); err != nil {
		return nil, err
	}
	return currencies, nil
}

// GetCurrency returns a currency by code (any enabled state).
func GetCurrency(code string) (*Currency, error) {
	cur := new(Currency)
	if err := database.Get(cur, "currencies/get", strings.ToUpper(strings.TrimSpace(code))); err != nil {
		return nil, ErrUnsupportedCurrency
	}
	return cur, nil
}

// ValidateCurrency returns the normalized code if it's supported and enabled,
// else ErrUnsupportedCurrency. Use this to guard any currency taken from input.
func ValidateCurrency(code string) (string, error) {
	cur, err := GetCurrency(code)
	if err != nil || !cur.Enabled {
		return "", ErrUnsupportedCurrency
	}
	return cur.Code, nil
}

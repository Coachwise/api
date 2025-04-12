package models

import (
	"context"

	database "github.com/socious-io/pkg_database"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type TokenBlacklist struct {
	ID        uuid.UUID `db:"id" json:"id"`
	Token     string    `db:"string" json:"token"`
	ExpiresAt string    `db:"expires_at" json:"expires_at"`
}

func (TokenBlacklist) TableName() string {
	return "tokens_blacklist"
}

func (TokenBlacklist) FetchQuery() string {
	return "tokens_blacklist/fetch"
}

func (tb *TokenBlacklist) Scan(rows *sqlx.Rows) error {
	return rows.StructScan(tb)
}

func (tb *TokenBlacklist) Create(ctx context.Context) error {
	rows, err := database.Query(
		ctx,
		"tokens_blacklist/create",
		tb.Token,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := tb.Scan(rows); err != nil {
			return err
		}
	}
	return nil
}

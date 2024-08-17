package auth

import (
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type OTP struct {
	ID        uuid.UUID `db:"id" json:"id"`
	UserID    uuid.UUID `db:"user_id" json:"user_id"`
	Code      int       `db:"code" json:"code"`
	ExpiresAt time.Time `db:"expired_at" json:"expired_at"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func (*OTP) TableName() string {
	return "otps"
}

func (*OTP) FetchQuery() string {
	return "otps/fetch"
}

func (o *OTP) Scan(rows *sqlx.Rows) error {
	return rows.StructScan(o)
}

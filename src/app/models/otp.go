package models

import (
	"context"
	"errors"
	"coachwise/src/logger"
	"math/rand/v2"
	"time"

	"coachwise/src/database"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// OTPCooldown is the minimum gap between codes for one user+purpose — stops a
// single phone from minting endless SMS.
const OTPCooldown = 2 * time.Minute

// ErrOTPCooldown means a code was requested again too soon.
var ErrOTPCooldown = errors.New("please wait before requesting another code")

type OTP struct {
	ID         uuid.UUID `db:"id" json:"id"`
	UserID     uuid.UUID `db:"user_id" json:"user_id"`
	Email      string    `db:"email" json:"email"`
	Code       int       `db:"code" json:"code"`
	Perpose    string    `db:"perpose" json:"perpose"`
	IsVerified bool      `db:"is_verified" json:"is_verified"`
	ExpiresAt  time.Time `db:"expired_at" json:"expired_at"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

func (OTP) TableName() string {
	return "otps"
}

func (OTP) FetchQuery() string {
	return "otps/fetch"
}

func (o *OTP) Scan(rows *sqlx.Rows) error {
	return rows.StructScan(o)
}

func (o *OTP) Create(ctx context.Context) error {

	rows, err := database.Query(
		ctx,
		"otp/create",
		o.UserID, o.Code, o.Perpose, o.Email,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := o.Scan(rows); err != nil {
			return err
		}
	}
	return nil
}

func (o *OTP) Verify(ctx context.Context) error {
	rows, err := database.Query(
		ctx,
		"otp/verify",
		o.UserID, o.Code,
	)

	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := o.Scan(rows); err != nil {
			return err
		}

	}
	return nil
}

func NewOTP(ctx context.Context, userID uuid.UUID, perpose string) (*OTP, error) {
	// Cooldown: refuse a new code if one was issued for this user+purpose within
	// the last OTPCooldown — cheap anti-spam per phone/account.
	if rows, err := database.Query(ctx, "otp/recent", userID, perpose, int(OTPCooldown.Seconds())); err == nil {
		recent := rows.Next()
		rows.Close()
		if recent {
			return nil, ErrOTPCooldown
		}
	}

	// Deleted accounts included: sending them a code is exactly how a returning
	// phone gets its account back.
	u, err := GetUserAny(userID)
	if err != nil {
		return nil, err
	}
	o := &OTP{
		UserID:  userID,
		Code:    int(100000 + rand.Float64()*900000),
		Perpose: perpose,
		Email:   u.Email,
	}
	if err := o.Create(ctx); err != nil {
		return nil, err
	}
	logger.Infof("OTP generated for %s code=%d\n", o.Email, o.Code)
	return o, nil
}

// ConsumeOTP validates the latest matching OTP for a user+purpose and marks it
// used in one atomic step. Returns true only when the code was valid, unexpired,
// and unused — so codes can't be replayed.
func ConsumeOTP(ctx context.Context, userID uuid.UUID, perpose string, code int) (bool, error) {
	rows, err := database.Query(ctx, "otp/consume", userID, perpose, code)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	return rows.Next(), nil
}

func GetOTPByUserID(user_id uuid.UUID) (*OTP, error) {
	o := new(OTP)
	if err := database.Get(o, "otp/fetch_by_userid", user_id); err != nil {
		return nil, err
	}
	return o, nil
}

func GetOTPByEmail(email string) (*OTP, error) {
	o := new(OTP)
	if err := database.Get(o, "otp/get_by_email", email); err != nil {
		return nil, err
	}
	return o, nil
}

package models

import (
	"context"

	"coachwise/src/database"

	"github.com/google/uuid"
)

// Device platforms (mirrors the table's CHECK).
const (
	PlatformAndroid = "android"
	PlatformIOS     = "ios"
	PlatformWeb     = "web"
)

// DeviceToken is one app install that can receive a push.
type DeviceToken struct {
	Token    string `db:"token" json:"token"`
	Platform string `db:"platform" json:"platform"`
}

func (DeviceToken) TableName() string { return "device_tokens" }

func ValidPlatform(p string) bool {
	switch p {
	case PlatformAndroid, PlatformIOS, PlatformWeb:
		return true
	}
	return false
}

// RegisterDeviceToken records or moves a token. Idempotent — the app calls it
// on every launch.
func RegisterDeviceToken(ctx context.Context, userID uuid.UUID, token, platform, locale string) error {
	var loc any
	if locale != "" {
		loc = locale
	}
	rows, err := database.Query(ctx, "device_tokens/create", userID, token, platform, loc)
	if err != nil {
		return err
	}
	rows.Close()
	return nil
}

func ListDeviceTokens(ctx context.Context, userID uuid.UUID) ([]DeviceToken, error) {
	items := []DeviceToken{}
	rows, err := database.Query(ctx, "device_tokens/list_by_user", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var d DeviceToken
		if err := rows.Scan(&d.Token, &d.Platform); err != nil {
			return nil, err
		}
		items = append(items, d)
	}
	return items, rows.Err()
}

// UnregisterDeviceToken drops one of the caller's own tokens (logout).
func UnregisterDeviceToken(ctx context.Context, token string, userID uuid.UUID) error {
	rows, err := database.Query(ctx, "device_tokens/delete", token, userID)
	if err != nil {
		return err
	}
	rows.Close()
	return nil
}

// DeleteDeviceToken drops a token the provider reported dead.
func DeleteDeviceToken(ctx context.Context, token string) error {
	rows, err := database.Query(ctx, "device_tokens/delete_token", token)
	if err != nil {
		return err
	}
	rows.Close()
	return nil
}

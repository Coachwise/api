package auth

import (
	"coachwise/src/config"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	ID      string `json:"id"`
	Refresh bool   `json:"refresh"`
	jwt.RegisteredClaims
}

// Access tokens are short-lived (a leaked one dies fast); refresh tokens are
// long and rotated on use, so day-to-day sessions survive without keeping a
// long-lived access token around.
const (
	AccessTokenTTL  = time.Hour
	RefreshTokenTTL = 30 * 24 * time.Hour
)

func GenerateToken(id string, refresh bool) (string, error) {
	ttl := AccessTokenTTL
	if refresh {
		ttl = RefreshTokenTTL
	}
	expirationTime := time.Now().Add(ttl)
	claims := &Claims{
		ID:      id,
		Refresh: refresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.Config.Secret))
}

func VerifyToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.Config.Secret), nil
	})
	if err != nil {
		return nil, err
	} else if !token.Valid {
		return nil, errors.New("invalid token")
	} else if claims, ok := token.Claims.(*Claims); ok {
		return claims, nil
	}

	return nil, errors.New("unknown claims type, cannot proceed")
}

func GenerateFullTokens(id string) (map[string]any, error) {
	accessToken, err := GenerateToken(id, false)
	if err != nil {
		return nil, err
	}
	refreshToken, err := GenerateToken(id, true)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
	}, nil
}

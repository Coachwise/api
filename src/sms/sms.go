// Package sms sends transactional SMS (OTP codes), routed to a provider by the
// recipient's country — mirroring the payments provider registry. Each gateway
// lives in its OWN file (kavenegar.go, sendgrid.go later); this file is only the
// interface + registry + dispatch. When no provider matches a number the code is
// just logged, which is how OTP works in dev until a gateway is configured.
package sms

import (
	"errors"
	"strings"

	"coachwise/src/config"
	"coachwise/src/logger"
)

// ErrSMS wraps a failure sending through a gateway.
var ErrSMS = errors.New("sms gateway error")

// Provider is one SMS gateway. Supports decides, from the phone's country, which
// numbers it handles.
type Provider interface {
	Name() string
	Supports(phone string) bool
	// SendOTP delivers a verification code (Kavenegar uses its OTP template).
	SendOTP(phone, code string) error
	// Send delivers a free-text message (for non-OTP use / SendGrid later).
	Send(phone, text string) error
}

var registry []Provider

// Init builds the provider registry from config (call once at startup, in both
// the API and the worker process since either may send). Each provider type is
// constructed by its own file's constructor.
func Init() {
	registry = nil
	for _, p := range config.Config.SMS.Providers {
		// An un-keyed provider is skipped so it doesn't swallow sends — the number
		// then falls through to logging (dev) instead of erroring every time.
		if strings.TrimSpace(p.APIKey) == "" {
			continue
		}
		switch strings.ToLower(p.Type) {
		case "kavenegar":
			registry = append(registry, newKavenegar(p.APIKey, p.Sender, p.Template, p.BaseURL, p.Countries))
		}
	}
}

func providerFor(phone string) Provider {
	for _, p := range registry {
		if p.Supports(phone) {
			return p
		}
	}
	return nil
}

// SendOTP delivers a verification code to a phone via the country's provider,
// falling back to logging when none matches (dev).
func SendOTP(phone, code string) error {
	if p := providerFor(phone); p != nil {
		return p.SendOTP(phone, code)
	}
	logger.Infof("SMS(otp) to %s: code=%s (no provider — logged)", phone, code)
	return nil
}

// Send delivers a free-text message, or logs it when no provider matches.
func Send(phone, text string) error {
	if p := providerFor(phone); p != nil {
		return p.Send(phone, text)
	}
	logger.Infof("SMS to %s: %s (no provider — logged)", phone, text)
	return nil
}

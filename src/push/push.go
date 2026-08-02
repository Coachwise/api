// Package push delivers mobile push through a provider registry (FCM today),
// like payments/ and sms/.
//
// Payloads carry NO text: the backend is English-only, so a push ships loc keys
// + args and the device resolves them from strings.xml / Localizable.strings.
package push

import (
	"context"
	"errors"
	"strings"

	"coachwise/src/config"
	"coachwise/src/logger"
)

// ErrDeadToken: the provider says this token is gone for good — delete the row.
var ErrDeadToken = errors.New("push: token no longer registered")

// Message is one notification as loc keys. Args must be strings (FCM and APNs
// both refuse anything else).
type Message struct {
	TitleLocKey  string
	TitleLocArgs []string
	BodyLocKey   string
	BodyLocArgs  []string
	Data         map[string]string // delivered to the tap handler, not shown
}

type sender interface {
	send(ctx context.Context, token string, m Message) error
}

var active sender

// Init selects the provider. Anything unusable leaves push disabled, so dev and
// the test suite run without Firebase credentials.
func Init() {
	provider := strings.ToLower(config.Config.Push.Provider)
	switch provider {
	case "":
		logger.Info("push: disabled (no provider configured)")
	case "fcm":
		s, err := newFCM()
		if err != nil {
			logger.Errorf("push: FCM init failed (%v) — push disabled", err)
			return
		}
		active = s
		logger.Infof("push: FCM ready (project %s)", s.projectID)
	default:
		logger.Errorf("push: unknown provider %q — push disabled", provider)
	}
}

func Enabled() bool { return active != nil }

// Deliver sends to every token and returns the ones the provider called dead.
// One stale device must not cost the user the notification on their others.
func Deliver(ctx context.Context, tokens []string, m Message) (dead []string) {
	if active == nil {
		return nil
	}
	for _, t := range tokens {
		err := active.send(ctx, t, m)
		switch {
		case err == nil:
		case errors.Is(err, ErrDeadToken):
			dead = append(dead, t)
		default:
			logger.Errorf("push: send failed: %v", err)
		}
	}
	return dead
}

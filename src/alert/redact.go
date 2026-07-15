package alert

import (
	"encoding/json"
	"fmt"
	"strings"
)

// A Discord channel is not a secure store: it has no expiry, it is searchable,
// and everyone in the org can read it. Anything that would let a reader take over
// an account must never reach it — a live OTP code and a phone number together
// ARE the login, since auth is passwordless (see the auth flow).
//
// Redaction is by key name and is deliberately blunt: an unknown key that merely
// looks sensitive is dropped. Losing a field costs a debugging round-trip;
// leaking one costs an account.
var secretKeys = map[string]bool{
	"password":         true,
	"current_password": true,
	"new_password":     true,
	"token":            true,
	"access_token":     true,
	"refresh_token":    true,
	"secret":           true,
	"code":             true, // OTP
	"otp":              true,
	"api_key":          true,
	"authorization":    true,
}

// identifierKeys are useful for tracing but must not be readable in full.
var identifierKeys = map[string]bool{
	"phone": true,
	"email": true,
}

// RedactBody takes a raw request body and returns something safe to post. Valid
// JSON is walked key by key; anything else is dropped entirely rather than
// guessed at, because a form-encoded or binary body could hide a secret anywhere.
func RedactBody(contentType string, raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	if !strings.Contains(contentType, "json") {
		return fmt.Sprintf("(%d bytes of %s — not shown)", len(raw), firstToken(contentType))
	}

	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return "(unparseable body — not shown)"
	}
	cleaned, err := json.Marshal(redactValue(v))
	if err != nil {
		return "(body could not be redacted — not shown)"
	}
	return string(cleaned)
}

func redactValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			key := strings.ToLower(k)
			switch {
			case secretKeys[key]:
				out[k] = "[redacted]"
			case identifierKeys[key]:
				if s, ok := val.(string); ok {
					out[k] = Mask(s)
				} else {
					out[k] = "[redacted]"
				}
			default:
				out[k] = redactValue(val)
			}
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = redactValue(val)
		}
		return out
	default:
		return v
	}
}

// Mask keeps just enough of an identifier to recognise it in a support thread —
// the last four characters — and hides the rest. "+989121234567" → "•••••••4567".
func Mask(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "••••"
	}
	return strings.Repeat("•", len(s)-4) + s[len(s)-4:]
}

// RedactHeaders keeps the headers worth having and drops the ones that carry
// credentials. Authorization is never included, not even partially: a bearer
// token prefix is still a token to anyone who can read the channel.
func RedactHeaders(get func(string) string) string {
	keep := []string{"User-Agent", "Referer", "X-Request-Id", "Content-Type"}
	var b strings.Builder
	for _, h := range keep {
		if v := get(h); v != "" {
			fmt.Fprintf(&b, "%s: %s\n", h, v)
		}
	}
	return strings.TrimSpace(b.String())
}

func firstToken(s string) string {
	if i := strings.Index(s, ";"); i > 0 {
		return s[:i]
	}
	if s == "" {
		return "unknown type"
	}
	return s
}

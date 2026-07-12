package sms

import "strings"

// Phone-number helpers shared by all SMS providers.

// digitsOnly strips '+', spaces, dashes and any non-digit.
func digitsOnly(phone string) string {
	var b strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// hasDialCode reports whether a phone belongs to one of the given dial codes
// (e.g. "98"). A bare local number (leading 0, no country) is treated as the
// first configured code — the app is Iran-first, so "09121234567" is Iran.
func hasDialCode(phone string, codes []string) bool {
	d := strings.TrimPrefix(digitsOnly(phone), "00") // 0098… → 98…
	for _, code := range codes {
		if strings.HasPrefix(d, code) {
			return true
		}
	}
	if len(codes) > 0 && strings.HasPrefix(d, "0") {
		return true
	}
	return false
}

// toLocalIran normalizes an Iranian number to Kavenegar's local receptor form
// "09XXXXXXXXX": +98/98/0098 prefixes → 0, a bare "9XXXXXXXXX" → prepend 0.
func toLocalIran(phone string) string {
	d := strings.TrimPrefix(digitsOnly(phone), "00")
	d = strings.TrimPrefix(d, "98")
	if !strings.HasPrefix(d, "0") {
		d = "0" + d
	}
	return d
}

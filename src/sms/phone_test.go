package sms

import "testing"

func TestHasDialCodeIran(t *testing.T) {
	iran := []string{"98"}
	cases := map[string]bool{
		"+989121234567":  true,  // +98
		"989121234567":   true,  // 98
		"0098912123456":  true,  // 0098
		"09121234567":    true,  // local
		"+1 415 555 2671": false, // US
		"447911123456":   false, // UK
	}
	for phone, want := range cases {
		if got := hasDialCode(phone, iran); got != want {
			t.Errorf("hasDialCode(%q) = %v, want %v", phone, got, want)
		}
	}
}

func TestToLocalIran(t *testing.T) {
	cases := map[string]string{
		"+989121234567": "09121234567",
		"989121234567":  "09121234567",
		"00989121234567": "09121234567",
		"09121234567":   "09121234567",
		"9121234567":    "09121234567",
	}
	for in, want := range cases {
		if got := toLocalIran(in); got != want {
			t.Errorf("toLocalIran(%q) = %q, want %q", in, got, want)
		}
	}
}

package utils

func StrPtr(val string) *string {
	if val == "" {
		return nil
	}
	return &val
}

// TruncateRunes shortens s to at most max runes (not bytes, so it never splits a
// multi-byte character — important for Persian), appending an ellipsis when cut.
func TruncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}

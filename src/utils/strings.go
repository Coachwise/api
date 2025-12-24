package utils

func StrPtr(val string) *string {
	if val == "" {
		return nil
	}
	return &val
}

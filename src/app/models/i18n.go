package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// LocalizedText is a translatable string stored as a JSONB map of locale → text,
// e.g. {"en": "Dead Hang", "fa": "آویزان ثابت"}. Used for exercise and category
// names/descriptions so a single row serves every language. A nil/empty map
// round-trips as `{}`.
type LocalizedText map[string]string

func (t *LocalizedText) Scan(value interface{}) error {
	if value == nil {
		*t = LocalizedText{}
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("failed to scan LocalizedText: %T", value)
	}
	if len(b) == 0 {
		*t = LocalizedText{}
		return nil
	}
	m := LocalizedText{}
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	*t = m
	return nil
}

func (t LocalizedText) Value() (driver.Value, error) {
	if t == nil {
		return "{}", nil
	}
	b, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// Get returns the text for locale, falling back to the given fallback (typically
// the plain default column) when the locale is missing.
func (t LocalizedText) Get(locale, fallback string) string {
	if v, ok := t[locale]; ok && v != "" {
		return v
	}
	return fallback
}

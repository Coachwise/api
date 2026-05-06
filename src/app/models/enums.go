package models

import (
	"database/sql/driver"
	"fmt"
)

type AttributeType string

// ENUM values
const (
	Text     AttributeType = "TEXT"
	Number   AttributeType = "NUMBER"
	Boolean  AttributeType = "BOOLEAN"
	Url      AttributeType = "URL"
	Datetime AttributeType = "DATETIME"
	Email    AttributeType = "EMAIL"
)

func (a *AttributeType) Scan(value interface{}) error {
	strValue, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan UserStatus: %v", value)
	}
	*a = AttributeType(strValue)
	return nil
}

func (a AttributeType) Value() (driver.Value, error) {
	return string(a), nil
}

type ExerciseSportType string

// ENUM values
const (
	SportTypeStrength ExerciseSportType = "STRENGTH"
	SportTypeClimbing ExerciseSportType = "CLIMBING"
	SportTypeCardio   ExerciseSportType = "CARDIO"
	SportTypeMobility ExerciseSportType = "MOBILITY"
	SportTypeGeneral  ExerciseSportType = "GENERAL"
)

func (e *ExerciseSportType) Scan(value interface{}) error {
	switch v := value.(type) {
	case string:
		*e = ExerciseSportType(v)
	case []byte:
		*e = ExerciseSportType(v)
	default:
		return fmt.Errorf("failed to scan ExerciseSportType: %v", value)
	}
	return nil
}

func (e ExerciseSportType) Value() (driver.Value, error) {
	return string(e), nil
}

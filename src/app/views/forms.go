package views

import (
	"time"
)

type ExerciseForm struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Public      bool   `json:"public"`
	Sets        []struct {
		Name     string         `json:"name"`
		RestTime time.Duration  `json:"rest_time"`
		RepCount *int           `json:"rep_count"`
		Duration *time.Duration `json:"duration"`
	} `json:"sets"`
}

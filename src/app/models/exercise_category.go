package models

import (
	"time"

	database "github.com/socious-io/pkg_database"

	"github.com/google/uuid"
)

// ExerciseCategory groups exercises for browsing. Translatable (name_i18n) and
// optionally scoped to a sport (sport_type NULL = any).
type ExerciseCategory struct {
	ID        uuid.UUID     `db:"id" json:"id"`
	Slug      string        `db:"slug" json:"slug"`
	NameI18n  LocalizedText `db:"name_i18n" json:"name_i18n"`
	SportType *string       `db:"sport_type" json:"sport_type"`
	SortOrder int           `db:"sort_order" json:"sort_order"`
	CreatedAt time.Time     `db:"created_at" json:"created_at"`
	UpdatedAt time.Time     `db:"updated_at" json:"updated_at"`
}

// ListExerciseCategories returns categories ordered for display. `sport` (an
// exercise_sport_type value, or "") keeps sport-scoped + any-sport categories.
func ListExerciseCategories(sport string) ([]ExerciseCategory, error) {
	cats := []ExerciseCategory{}
	if err := database.QuerySelect("exercise_categories/list", &cats, sport); err != nil {
		return nil, err
	}
	return cats, nil
}

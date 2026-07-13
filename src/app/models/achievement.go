package models

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"coachwise/src/database"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx/types"
)

var ErrAchievementNotFound = errors.New("achievement not found")

// Achievement is a coach-granted badge on an athlete's profile.
type Achievement struct {
	ID          uuid.UUID `db:"id" json:"id"`
	AthleteID   uuid.UUID `db:"athlete_id" json:"athlete_id"`
	CoachID     uuid.UUID `db:"coach_id" json:"coach_id"`
	Title       string    `db:"title" json:"title"`
	Description *string   `db:"description" json:"description,omitempty"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// PersonalRecord is a derived PR for an athlete on an exercise (best per metric,
// from APPROVED test submissions). Unset metrics are nil.
type PersonalRecord struct {
	ExerciseID   uuid.UUID `db:"exercise_id" json:"exercise_id"`
	ExerciseName string    `db:"exercise_name" json:"exercise_name"`
	BestWeight   *float64  `db:"best_weight" json:"best_weight,omitempty"`
	BestReps     *int      `db:"best_reps" json:"best_reps,omitempty"`
	BestTime     *int      `db:"best_time" json:"best_time,omitempty"`
}

func GrantAchievement(ctx context.Context, a *Achievement) error {
	rows, err := database.Query(ctx, "achievements/create", a.AthleteID, a.CoachID, a.Title, a.Description)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(a); err != nil {
			return err
		}
	}
	return nil
}

func ListAchievements(ctx context.Context, athleteID uuid.UUID) ([]Achievement, error) {
	items := []Achievement{}
	if err := database.QuerySelect("achievements/list_by_athlete", &items, athleteID); err != nil {
		return nil, err
	}
	return items, nil
}

func DeleteAchievement(ctx context.Context, id, coachID uuid.UUID) error {
	rows, err := database.Query(ctx, "achievements/delete", id, coachID)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return ErrAchievementNotFound
	}
	return nil
}

// ListPersonalRecords returns an athlete's PRs derived from submitted test logs.
func ListPersonalRecords(ctx context.Context, athleteID uuid.UUID) ([]PersonalRecord, error) {
	items := []PersonalRecord{}
	if err := database.QuerySelect("achievements/prs", &items, athleteID); err != nil {
		return nil, err
	}
	return items, nil
}

// AchievementLayout is a user's curated ordering + visibility of their profile
// achievements. Items are keyed "badge:<id>" or "record:<exercise_id>".
type AchievementLayout struct {
	Order  []string `json:"order"`
	Hidden []string `json:"hidden"`
}

// GetAchievementLayout returns a user's saved profile layout (empty if unset).
func GetAchievementLayout(ctx context.Context, userID uuid.UUID) (*AchievementLayout, error) {
	layout := &AchievementLayout{Order: []string{}, Hidden: []string{}}
	rows, err := database.Query(ctx, "achievements/layout_get", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		var raw types.JSONText
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, layout); err != nil {
				return nil, err
			}
		}
	}
	if layout.Order == nil {
		layout.Order = []string{}
	}
	if layout.Hidden == nil {
		layout.Hidden = []string{}
	}
	return layout, nil
}

// SetAchievementLayout upserts a user's profile layout.
func SetAchievementLayout(ctx context.Context, userID uuid.UUID, layout AchievementLayout) error {
	raw, err := json.Marshal(layout)
	if err != nil {
		return err
	}
	rows, err := database.Query(ctx, "achievements/layout_set", userID, raw)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return errors.New("failed to save layout")
	}
	return nil
}

// CountActiveClients returns how many distinct clients hold an active package
// subscription with the coach.
func CountActiveClients(ctx context.Context, coachID uuid.UUID) (int, error) {
	var n int
	rows, err := database.Query(ctx, "subscriptions/active_client_count", coachID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if rows.Next() {
		if err := rows.Scan(&n); err != nil {
			return 0, err
		}
	}
	return n, nil
}

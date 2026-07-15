package models

import (
	"context"
	"errors"
	"time"

	"coachwise/src/database"

	"github.com/google/uuid"
)

var (
	ErrWorkoutLogNotFound = errors.New("workout log not found")
)

type WorkoutLog struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	SessionID       *uuid.UUID `db:"session_id" json:"session_id"`
	TestRequestID   *uuid.UUID `db:"test_request_id" json:"test_request_id,omitempty"`
	ExerciseID      *uuid.UUID `db:"exercise_id" json:"exercise_id"`
	ExerciseName    *string    `db:"exercise_name" json:"exercise_name"`
	SetNumber       int        `db:"set_number" json:"set_number"`
	Reps            *int       `db:"reps" json:"reps"`
	Weight          *float64   `db:"weight" json:"weight"`
	RPE             *float64   `db:"rpe" json:"rpe"`
	DurationSeconds *int       `db:"duration_seconds" json:"duration_seconds"`
	Grade           *string    `db:"grade" json:"grade"`
	Completed       bool       `db:"completed" json:"completed"`
	Attempts        *int       `db:"attempts" json:"attempts"`
	Notes           *string    `db:"notes" json:"notes"`
	LoggedAt        time.Time  `db:"logged_at" json:"-"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	Tags            []Tag      `db:"-" json:"tags"`
	DeletedAt *time.Time `db:"deleted_at" json:"-"`
}

func (WorkoutLog) TableName() string {
	return "workout_logs"
}

func (WorkoutLog) FetchQuery() string {
	return "workout_logs/fetch"
}

func (wl *WorkoutLog) Create(ctx context.Context) error {
	rows, err := database.Query(
		ctx,
		"workout_logs/create",
		wl.SessionID, wl.ExerciseID, wl.ExerciseName, wl.SetNumber,
		wl.Reps, wl.Weight, wl.RPE, wl.DurationSeconds,
		wl.Grade, wl.Completed, wl.Attempts, wl.Notes,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(wl); err != nil {
			return err
		}
	}
	return nil
}

func (wl *WorkoutLog) Update(ctx context.Context) error {
	rows, err := database.Query(
		ctx,
		"workout_logs/update",
		wl.ID, wl.Reps, wl.Weight, wl.RPE, wl.DurationSeconds,
		wl.Grade, wl.Completed, wl.Attempts, wl.Notes,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(wl); err != nil {
			return err
		}
	}
	return nil
}

func GetWorkoutLog(id uuid.UUID) (*WorkoutLog, error) {
	wl := new(WorkoutLog)
	if err := database.Fetch(wl, id); err != nil {
		return nil, err
	}
	return wl, nil
}

func GetSessionWorkoutLogs(ctx context.Context, sessionID uuid.UUID) ([]WorkoutLog, error) {
	var (
		logs      []WorkoutLog
		fetchList []database.FetchList
		ids       []interface{}
	)

	if err := database.QuerySelect("workout_logs/fetch_by_session", &fetchList, sessionID); err != nil {
		return nil, err
	}

	if len(fetchList) < 1 {
		return logs, nil
	}

	for _, f := range fetchList {
		ids = append(ids, f.ID)
	}

	if err := database.Fetch(&logs, ids...); err != nil {
		return nil, err
	}

	return logs, nil
}

func DeleteWorkoutLog(ctx context.Context, id, userID uuid.UUID) error {
	rows, err := database.Query(ctx, "workout_logs/delete", id, userID)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return ErrWorkoutLogNotFound
	}
	return nil
}

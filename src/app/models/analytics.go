package models

import (
	"context"

	"coachwise/src/database"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type DailyAnalytics struct {
	Date               string         `db:"date" json:"date"`
	SessionsCount      int            `db:"sessions_count" json:"sessions_count"`
	TotalDuration      *float64       `db:"total_duration" json:"total_duration"` // minutes
	PlansCompleted     pq.StringArray `db:"plans_completed" json:"plans_completed"`
	ExercisesCompleted int            `db:"exercises_completed" json:"exercises_completed"`
	TotalSets          int            `db:"total_sets" json:"total_sets"`
	TotalReps          *int           `db:"total_reps" json:"total_reps"`
	TotalVolume        *float64       `db:"total_volume" json:"total_volume"`
	TotalCount         int            `db:"total_count" json:"-"` // For pagination
}

func ListDailyAnalytics(ctx context.Context, userID uuid.UUID, p database.Paginate) ([]DailyAnalytics, int, error) {
	var (
		items      = []DailyAnalytics{}
		totalCount int
	)

	if err := database.QuerySelect("sessions/analytics", &items, userID, p.Limit, p.Offset); err != nil {
		return nil, 0, err
	}

	if len(items) > 0 {
		totalCount = items[0].TotalCount
	}

	return items, totalCount, nil
}

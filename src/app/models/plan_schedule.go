package models

import (
	"context"
	"time"

	"coachwise/src/database"

	"github.com/google/uuid"
)

type PlanScheduleStatus string

const (
	PlanScheduleStatusActive   PlanScheduleStatus = "ACTIVE"
	PlanScheduleStatusCanceled PlanScheduleStatus = "CANCELED"
)

type PlanSchedule struct {
	ID           uuid.UUID          `db:"id" json:"id"`
	UserID       uuid.UUID          `db:"user_id" json:"user_id"`
	PlanID       uuid.UUID          `db:"plan_id" json:"plan_id"`
	ScheduledFor time.Time          `db:"scheduled_for" json:"scheduled_for"`
	Status       PlanScheduleStatus `db:"status" json:"status"`
	Notes        *string            `db:"notes" json:"notes"`
	CreatedAt    time.Time          `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `db:"updated_at" json:"updated_at"`
}

func (PlanSchedule) TableName() string {
	return "plan_schedules"
}

func (PlanSchedule) FetchQuery() string {
	return "plans/schedule/fetch"
}

func CreatePlanSchedule(ctx context.Context, ps *PlanSchedule) error {
	rows, err := database.Query(
		ctx,
		"plans/schedule/create",
		ps.UserID,
		ps.PlanID,
		ps.ScheduledFor,
		ps.Notes,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(ps); err != nil {
			return err
		}
	}
	return nil
}

func ListPlanSchedule(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]PlanSchedule, error) {
	var items []PlanSchedule
	if err := database.QuerySelect("plans/schedule/list_by_range", &items, userID, from, to); err != nil {
		return nil, err
	}
	return items, nil
}

func ListPlanSchedulePaginated(ctx context.Context, userID uuid.UUID, p database.Paginate) ([]PlanSchedule, int, error) {
	var (
		items     = []PlanSchedule{}
		fetchList []database.FetchList
		ids       []interface{}
		total     int
	)

	if err := database.QuerySelect("plans/schedule/list", &fetchList, userID, p.Limit, p.Offset); err != nil {
		return nil, 0, err
	}

	if len(fetchList) < 1 {
		return items, 0, nil
	}

	total = fetchList[0].TotalCount

	for _, f := range fetchList {
		ids = append(ids, f.ID)
	}

	if err := database.Fetch(&items, ids...); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func UpdatePlanSchedule(ctx context.Context, ps *PlanSchedule) error {
	rows, err := database.Query(
		ctx,
		"plans/schedule/update",
		ps.ID,
		ps.Status,
		ps.Notes,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(ps); err != nil {
			return err
		}
	}
	return nil
}

func DeletePlanSchedule(ctx context.Context, id uuid.UUID) error {
	rows, err := database.Query(ctx, "plans/schedule/delete", id)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return ErrPlanNotFound
	}
	return nil
}

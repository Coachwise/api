package models

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"coachwise/src/database"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx/types"
)

var (
	ErrPlanNotFound     = errors.New("plan not found")
	ErrPlanAccessDenied = errors.New("plan not accessible")
)

type Plan struct {
	ID            uuid.UUID `db:"id" json:"id"`
	UserID        uuid.UUID `db:"user_id" json:"user_id"`
	Public        bool      `db:"public" json:"public"`
	Name          string    `db:"name" json:"name"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
	ExerciseCount int       `db:"exercise_count" json:"exercise_count"`
	// EstimatedSeconds is a rough completion estimate: per set, the work time
	// (set duration, or rep_count × 6s for rep-based sets) plus its rest time.
	EstimatedSeconds int `db:"estimated_seconds" json:"estimated_seconds"`
	// User is the plan owner (the coach, for an assigned plan), populated from the
	// user_id FK via row_to_json. UserJson is auto-unmarshalled into User by
	// godatabase (db tag "user" ↔ json tag "user").
	// json tag must be exactly "user" (no ,omitempty) so godatabase matches the
	// UserJson db:"user" field to it and unmarshals the owner into User.
	User     *User          `db:"-" json:"user"`
	UserJson types.JSONText `db:"user" json:"-"`
	// Absorbs the generated tsvector from `SELECT p.*`; not serialized.
	SearchVector *string `db:"search_vector" json:"-"`
	DeletedAt *time.Time `db:"deleted_at" json:"-"`
}

type PlanExercise struct {
	ID            uuid.UUID      `db:"id" json:"id"`
	ExerciseID    uuid.UUID      `db:"exercise_id" json:"exercise_id"`
	PlanID        uuid.UUID      `db:"plan_id" json:"plan_id"`
	ExerciseOrder int            `db:"exercise_order" json:"exercise_order"`
	RestTime      time.Duration  `db:"rest_time" json:"rest_time"`
	Intensity     int            `db:"intensity" json:"intensity"` // 1-10 scale
	CreatedAt     time.Time      `db:"created_at" json:"created_at"`
	Exercise      types.JSONText `db:"exercise" json:"exercise"`
}

type PlanAssignee struct {
	ID        uuid.UUID  `db:"id" json:"id"`
	PlanID    uuid.UUID  `db:"plan_id" json:"plan_id"`
	UserID    uuid.UUID  `db:"user_id" json:"user_id"`
	Assigner  uuid.UUID  `db:"assigner" json:"assigner"`
	PackageID *uuid.UUID `db:"package_id" json:"package_id,omitempty"` // set when assigned via a package; NULL = manual
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
}

func (Plan) TableName() string {
	return "plans"
}

func (Plan) FetchQuery() string {
	return "plans/fetch"
}

func (p *Plan) Create(ctx context.Context) error {
	rows, err := database.Query(
		ctx,
		"plans/create",
		p.UserID, p.Public, p.Name,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(p); err != nil {
			return err
		}
	}
	return nil
}

func (p *Plan) Update(ctx context.Context) error {
	rows, err := database.Query(
		ctx,
		"plans/update",
		p.ID, p.Name, p.Public,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(p); err != nil {
			return err
		}
	}
	return nil
}

func DeletePlan(ctx context.Context, id uuid.UUID) error {
	rows, err := database.Query(ctx, "plans/delete", id)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return ErrPlanNotFound
	}
	return nil
}

func CountPersonalPlans(ctx context.Context, userID uuid.UUID) (int, error) {
	var counts []struct {
		Count int `db:"count"`
	}
	if err := database.QuerySelect("plans/count_personal", &counts, userID); err != nil {
		return 0, err
	}
	if len(counts) == 0 {
		return 0, nil
	}
	return counts[0].Count, nil
}

func ListPlansPaginated(ctx context.Context, userID uuid.UUID, onlyPublic bool, search string, p database.Paginate) ([]Plan, int, error) {
	plans := []Plan{}
	var (
		fetchList []database.FetchList
		total     int
	)
	if err := database.QuerySelect("plans/list", &fetchList, userID, onlyPublic, toTSQueryPrefix(search), p.Limit, p.Offset); err != nil {
		return nil, 0, err
	}
	if len(fetchList) == 0 {
		return plans, 0, nil
	}
	total = fetchList[0].TotalCount
	// Fetch hydrates the rows via plans/fetch (incl. the owner user) and
	// auto-unmarshals the json fields.
	args := make([]interface{}, len(fetchList))
	for i, f := range fetchList {
		args[i] = f.ID
	}
	if err := database.Fetch(&plans, args...); err != nil {
		return nil, 0, err
	}
	return plans, total, nil
}

func GetPlanForUser(ctx context.Context, planID, userID uuid.UUID) (*Plan, error) {
	p := new(Plan)
	if err := database.Get(p, "plans/get_for_user", planID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPlanAccessDenied
		}
		return nil, err
	}
	return p, nil
}

func AddPlanExercise(ctx context.Context, pe *PlanExercise) error {
	rows, err := database.Query(
		ctx,
		"plans/exercises/create",
		pe.ExerciseID, pe.PlanID, pe.ExerciseOrder, pe.RestTime, pe.Intensity,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(pe); err != nil {
			return err
		}
	}
	return nil
}

func ListPlanExercises(ctx context.Context, planID uuid.UUID) ([]PlanExercise, error) {
	var list []PlanExercise
	if err := database.QuerySelect("plans/exercises/fetch_by_plan", &list, planID); err != nil {
		return nil, err
	}
	return list, nil
}

func RemovePlanExercise(ctx context.Context, planID, exerciseID uuid.UUID) error {
	rows, err := database.Query(ctx, "plans/exercises/delete", planID, exerciseID)
	if err != nil {
		return err
	}
	rows.Close()
	return nil
}

func AssignPlan(ctx context.Context, assignee *PlanAssignee) error {
	rows, err := database.Query(
		ctx,
		"plans/assignees/create",
		assignee.PlanID, assignee.UserID, assignee.Assigner, assignee.PackageID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(assignee); err != nil {
			return err
		}
	}
	return nil
}

func ListPlanAssignees(ctx context.Context, planID uuid.UUID) ([]PlanAssignee, error) {
	var list []PlanAssignee
	if err := database.QuerySelect("plans/assignees/list_by_plan", &list, planID); err != nil {
		return nil, err
	}
	return list, nil
}

func UnassignPlan(ctx context.Context, planID, userID uuid.UUID) error {
	rows, err := database.Query(ctx, "plans/assignees/delete", planID, userID)
	if err != nil {
		return err
	}
	rows.Close()
	return nil
}

func ListUserAssignedPlans(ctx context.Context, userID uuid.UUID) ([]Plan, error) {
	var plans []Plan
	if err := database.QuerySelect("plans/list_assigned", &plans, userID); err != nil {
		return nil, err
	}
	return plans, nil
}

package models

import (
	"context"
	"database/sql"
	"errors"
	"time"

	database "github.com/socious-io/pkg_database"

	"github.com/google/uuid"
)

var (
	ErrPlanNotFound     = errors.New("plan not found")
	ErrPlanAccessDenied = errors.New("plan not accessible")
)

type Plan struct {
	ID        uuid.UUID `db:"id" json:"id"`
	UserID    uuid.UUID `db:"user_id" json:"user_id"`
	Public    bool      `db:"public" json:"public"`
	Name      string    `db:"name" json:"name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type PlanExercise struct {
	ID            uuid.UUID     `db:"id" json:"id"`
	ExerciseID    uuid.UUID     `db:"exercise_id" json:"exercise_id"`
	PlanID        uuid.UUID     `db:"plan_id" json:"plan_id"`
	ExerciseOrder int           `db:"exercise_order" json:"exercise_order"`
	RestTime      time.Duration `db:"rest_time" json:"rest_time"`
	CreatedAt     time.Time     `db:"created_at" json:"created_at"`
}

type PlanAssignee struct {
	ID        uuid.UUID `db:"id" json:"id"`
	PlanID    uuid.UUID `db:"plan_id" json:"plan_id"`
	UserID    uuid.UUID `db:"user_id" json:"user_id"`
	Assigner  uuid.UUID `db:"assigner" json:"assigner"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
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
	db := database.GetDB()
	res, err := db.ExecContext(ctx, "DELETE FROM plans WHERE id=$1", id)
	if err != nil {
		return err
	}
	if count, _ := res.RowsAffected(); count == 0 {
		return ErrPlanNotFound
	}
	return nil
}

func CountPersonalPlans(ctx context.Context, userID uuid.UUID) (int, error) {
	db := database.GetDB()
	var count int
	if err := db.GetContext(ctx, &count, "SELECT COUNT(*) FROM plans WHERE user_id=$1 AND public=false", userID); err != nil {
		return 0, err
	}
	return count, nil
}

func ListPlans(ctx context.Context, userID uuid.UUID, onlyPublic bool) ([]Plan, error) {
	var plans []Plan
	db := database.GetDB()
	query := `
		SELECT DISTINCT p.*
		FROM plans p
		LEFT JOIN plan_assignees pa ON pa.plan_id = p.id
		WHERE p.user_id = $1
		   OR pa.user_id = $1
		   OR p.public = true`
	args := []interface{}{userID}
	if onlyPublic {
		query = `
			SELECT DISTINCT p.*
			FROM plans p
			WHERE p.public = true`
		args = []interface{}{}
	}
	if err := db.SelectContext(ctx, &plans, query, args...); err != nil {
		return nil, err
	}
	return plans, nil
}

func GetPlanForUser(ctx context.Context, planID, userID uuid.UUID) (*Plan, error) {
	p := new(Plan)
	db := database.GetDB()
	err := db.GetContext(ctx, p, `
		SELECT p.*
		FROM plans p
		WHERE p.id = $1 AND (
			p.user_id = $2
			OR p.public = true
			OR EXISTS (SELECT 1 FROM plan_assignees pa WHERE pa.plan_id = p.id AND pa.user_id = $2)
		)`, planID, userID)
	if err != nil {
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
		pe.ExerciseID, pe.PlanID, pe.ExerciseOrder, pe.RestTime,
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
	db := database.GetDB()
	_, err := db.ExecContext(ctx, "DELETE FROM plan_exercises WHERE plan_id=$1 AND exercise_id=$2", planID, exerciseID)
	return err
}

func AssignPlan(ctx context.Context, assignee *PlanAssignee) error {
	rows, err := database.Query(
		ctx,
		"plans/assignees/create",
		assignee.PlanID, assignee.UserID, assignee.Assigner,
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
	db := database.GetDB()
	_, err := db.ExecContext(ctx, "DELETE FROM plan_assignees WHERE plan_id=$1 AND user_id=$2", planID, userID)
	return err
}

func ListUserAssignedPlans(ctx context.Context, userID uuid.UUID) ([]Plan, error) {
	var plans []Plan
	if err := database.QuerySelect("plans/list_assigned", &plans, userID); err != nil {
		return nil, err
	}
	return plans, nil
}

package models

import (
	"context"
	"time"

	database "github.com/socious-io/pkg_database"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx/types"
)

type Exercise struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	UserID      *uuid.UUID `json:"user_id" db:"user_id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description" db:"description"`
	Public      bool       `json:"public" db:"public"`
	Sets        []Set      `json:"sets" db:"-"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`

	SetsJson types.JSONText `db:"sets" json:"-"`
}

type Set struct {
	ID         uuid.UUID      `json:"id" db:"id"`
	Name       *string        `json:"name" db:"name"`
	ExerciseID uuid.UUID      `json:"exercise_id" db:"exercise_id"`
	SetNumber  int            `json:"set_number" db:"set_number"`
	RestTime   time.Duration  `json:"rest_time" db:"rest_time"`
	RepCount   *int           `json:"rep_count" db:"rep_count"`
	Duration   *time.Duration `json:"duration" db:"duration"`
	CreatedAt  time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at" db:"updated_at"`
}

func (*Exercise) TableName() string {
	return "exercises"
}

func (*Exercise) FetchQuery() string {
	return "exercises/fetch"
}

func (e *Exercise) Create(ctx context.Context) error {
	tx, err := database.GetDB().Beginx()
	if err != nil {
		return err
	}
	rows, err := database.TxQuery(
		ctx,
		tx,
		"exercises/create",
		e.UserID, e.Name, e.Description, e.Public,
	)
	if err != nil {
		tx.Rollback()
		return err
	}
	for rows.Next() {
		if err := rows.StructScan(e); err != nil {
			tx.Rollback()
			return err
		}
	}
	rows.Close()

	for i := range e.Sets {
		e.Sets[i].ExerciseID = e.ID
		e.Sets[i].SetNumber = i + 1
	}

	if _, err := database.TxExecuteQuery(tx, "exercises/create_sets", e.Sets); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return database.Fetch(e, e.ID)
}

func (e *Exercise) Update(ctx context.Context) error {
	tx, err := database.GetDB().Beginx()
	if err != nil {
		return err
	}
	rows, err := database.TxQuery(
		ctx,
		tx,
		"exercises/update",
		e.ID, e.Name, e.Description, e.Public,
	)
	if err != nil {
		tx.Rollback()
		return err
	}
	for rows.Next() {
		if err := rows.StructScan(e); err != nil {
			tx.Rollback()
			return err
		}
	}
	rows.Close()

	for i := range e.Sets {
		e.Sets[i].SetNumber = i + 1
	}

	if _, err := database.TxExecuteQuery(tx, "exercises/update_sets", e.Sets); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return database.Fetch(e, e.ID)
}

func (*Set) TableName() string {
	return "sets"
}

func (*Set) FetchQuery() string {
	return "exercises/fetch_sets"
}

func GetExrcise(id uuid.UUID) (*Exercise, error) {
	e := new(Exercise)
	if err := database.Fetch(e, id); err != nil {
		return nil, err
	}
	return e, nil
}

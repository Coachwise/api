package models

import (
	"coachwise/src/database"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Exercise struct {
	ID        uuid.UUID     `json:"id"`
	UserID    uuid.NullUUID `json:"user_id"`
	Name      *string       `json:"name"`
	Public    sql.NullBool  `json:"public"`
	Sets      []Set         `json:"sets"`
	CreatedAt sql.NullTime  `json:"created_at"`
	UpdatedAt sql.NullTime  `json:"updated_at"`
}

type Set struct {
	ID        uuid.UUID     `json:"id"`
	Name      *string       `json:"name"`
	SetNumber int           `json:"set_number"`
	RestTime  time.Duration `json:"rest_time"`
	Reps      []Rep         `json:"reps"`
}

type Rep struct {
	ID       uuid.UUID     `json:"id"`
	RepCount sql.NullInt64 `json:"rep_count"`
	Duration time.Duration `json:"duration"`
	RestTime time.Duration `json:"rest_time"`
}

func (*Exercise) TableName() string {
	return "exercises"
}

func (*Exercise) FetchQuery() string {
	return "exercises/fetch"
}

func (e *Exercise) Scan(rows *sqlx.Rows) error {
	return rows.StructScan(e)
}

func NewExrcise(id string) (*Exercise, error) {
	e := new(Exercise)
	if err := database.Get(e, id); err != nil {
		return nil, err
	}
	return e, nil
}

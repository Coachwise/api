package models

import (
	"context"
	"time"

	"coachwise/src/database"

	"github.com/google/uuid"
)

type Session struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	UserID      uuid.UUID  `db:"user_id" json:"user_id"`
	SessionType string     `db:"session_type" json:"session_type"`
	PlanID      *uuid.UUID `db:"plan_id" json:"plan_id"`
	Status      string     `db:"status" json:"status"`
	StartedAt   time.Time  `db:"started_at" json:"started_at"`
	EndedAt     *time.Time `db:"ended_at" json:"ended_at"`
	Notes       *string    `db:"notes" json:"notes"`
	Intensity   *int       `db:"intensity" json:"intensity"`
	Quality     *int       `db:"quality" json:"quality"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

func (Session) TableName() string {
	return "sessions"
}

func (Session) FetchQuery() string {
	return "sessions/fetch"
}

func (s *Session) Create(ctx context.Context) error {
	rows, err := database.Query(
		ctx,
		"sessions/create",
		s.UserID, s.SessionType, s.PlanID, s.Status, s.StartedAt, s.Notes,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(s); err != nil {
			return err
		}
	}
	return nil
}

func (s *Session) Update(ctx context.Context) error {
	rows, err := database.Query(
		ctx,
		"sessions/update",
		s.ID, s.Status, s.EndedAt, s.Notes, s.Intensity, s.Quality,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(s); err != nil {
			return err
		}
	}
	return nil
}

func GetSession(id uuid.UUID) (*Session, error) {
	s := new(Session)
	if err := database.Fetch(s, id); err != nil {
		return nil, err
	}
	return s, nil
}

func GetActiveSessions(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	var (
		sessions  []Session
		fetchList []database.FetchList
		ids       []interface{}
	)

	if err := database.QuerySelect("sessions/fetch_active", &fetchList, userID); err != nil {
		return nil, err
	}

	if len(fetchList) < 1 {
		return sessions, nil
	}

	for _, f := range fetchList {
		ids = append(ids, f.ID)
	}

	if err := database.Fetch(&sessions, ids...); err != nil {
		return nil, err
	}

	return sessions, nil
}

func GetUserSessions(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	var (
		sessions  []Session
		fetchList []database.FetchList
		ids       []interface{}
	)

	if err := database.QuerySelect("sessions/fetch_by_user", &fetchList, userID); err != nil {
		return nil, err
	}

	if len(fetchList) < 1 {
		return sessions, nil
	}

	for _, f := range fetchList {
		ids = append(ids, f.ID)
	}

	if err := database.Fetch(&sessions, ids...); err != nil {
		return nil, err
	}

	return sessions, nil
}

func GetUserSessionsPaginated(ctx context.Context, userID uuid.UUID, paginate database.Paginate) ([]Session, int, error) {
	var (
		sessions  []Session
		fetchList []database.FetchList
		ids       []interface{}
		total     int
	)

	if err := database.QuerySelect("sessions/list", &fetchList, userID, paginate.Limit, paginate.Offset); err != nil {
		return nil, 0, err
	}

	if len(fetchList) < 1 {
		return sessions, 0, nil
	}

	total = fetchList[0].TotalCount

	for _, f := range fetchList {
		ids = append(ids, f.ID)
	}

	if err := database.Fetch(&sessions, ids...); err != nil {
		return nil, 0, err
	}

	return sessions, total, nil
}

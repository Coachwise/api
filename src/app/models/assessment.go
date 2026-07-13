package models

import (
	"context"
	"errors"
	"time"

	"coachwise/src/database"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx/types"
)

var (
	ErrTestNotFound     = errors.New("test not found")
	ErrTestAccessDenied = errors.New("test not accessible")
)

// Test is a coach-built assessment: a set of exercises, each measured by a unit.
type Test struct {
	ID          uuid.UUID `db:"id" json:"id"`
	CoachID     uuid.UUID `db:"coach_id" json:"coach_id"`
	Name        string    `db:"name" json:"name"`
	Description *string   `db:"description" json:"description,omitempty"`
	Public      bool      `db:"public" json:"public"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
	// Hydrated in fetch.sql (no N+1).
	ItemCount int            `db:"item_count" json:"item_count"`
	Items     types.JSONText `db:"items" json:"items"`
	// The owner, as a full user (so assigned athletes see who set the protocol).
	Coach     *User          `db:"-" json:"coach"`
	CoachJson types.JSONText `db:"coach" json:"-"`
}

// TestItemInput is one exercise in a test plus which metrics to measure. An item
// may track a combination (e.g. a weighted pull-up tracks reps AND weight).
type TestItemInput struct {
	ExerciseID  uuid.UUID `json:"exercise_id" binding:"required"`
	TrackReps   bool      `json:"track_reps"`
	TrackWeight bool      `json:"track_weight"`
	TrackTime   bool      `json:"track_time"`
	TargetValue *float64  `json:"target_value"`
}

// TestItemRow is the stored item, used to resolve a submission's metrics+exercise.
type TestItemRow struct {
	ID          uuid.UUID `db:"id"`
	ExerciseID  uuid.UUID `db:"exercise_id"`
	TrackReps   bool      `db:"track_reps"`
	TrackWeight bool      `db:"track_weight"`
	TrackTime   bool      `db:"track_time"`
}

func (Test) TableName() string  { return "tests" }
func (Test) FetchQuery() string { return "tests/fetch" }

func (t *Test) Create(ctx context.Context) error {
	rows, err := database.Query(ctx, "tests/create", t.CoachID, t.Name, t.Description, t.Public)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(t); err != nil {
			return err
		}
	}
	return nil
}

func (t *Test) Update(ctx context.Context) error {
	rows, err := database.Query(ctx, "tests/update", t.ID, t.Name, t.Description, t.Public)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(t); err != nil {
			return err
		}
	}
	return nil
}

func GetTest(ctx context.Context, id uuid.UUID) (*Test, error) {
	t := new(Test)
	if err := database.Fetch(t, id); err != nil {
		return nil, ErrTestNotFound
	}
	return t, nil
}

// GetTestForCoach returns a test only when it belongs to the given coach.
func GetTestForCoach(ctx context.Context, id, coachID uuid.UUID) (*Test, error) {
	t, err := GetTest(ctx, id)
	if err != nil {
		return nil, err
	}
	if t.CoachID != coachID {
		return nil, ErrTestAccessDenied
	}
	return t, nil
}

func DeleteTest(ctx context.Context, id uuid.UUID) error {
	rows, err := database.Query(ctx, "tests/delete", id)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return ErrTestNotFound
	}
	return nil
}

func ListCoachTestsPaginated(ctx context.Context, coachID uuid.UUID, p database.Paginate) ([]Test, int, error) {
	var (
		items     = []Test{}
		fetchList []database.FetchList
		ids       []interface{}
		total     int
	)
	if err := database.QuerySelect("tests/list", &fetchList, coachID, p.Limit, p.Offset); err != nil {
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

// ListAssignedTestsPaginated returns the protocols a coach has assigned to an
// athlete (so the athlete can run them like their own).
func ListAssignedTestsPaginated(ctx context.Context, athleteID uuid.UUID, p database.Paginate) ([]Test, int, error) {
	var (
		items     = []Test{}
		fetchList []database.FetchList
		ids       []interface{}
		total     int
	)
	if err := database.QuerySelect("tests/list_assigned", &fetchList, athleteID, p.Limit, p.Offset); err != nil {
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

// IsTestAssignedTo reports whether a coach has assigned this test to the athlete.
func IsTestAssignedTo(ctx context.Context, testID, athleteID uuid.UUID) (bool, error) {
	rows, err := database.Query(ctx, "tests/check_assigned", testID, athleteID)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	return rows.Next(), nil
}

// CanRunTest reports whether the user may run/view a protocol: they own it, or a
// coach has assigned it to them.
func CanRunTest(ctx context.Context, testID, userID uuid.UUID) (bool, error) {
	if _, err := GetTestForCoach(ctx, testID, userID); err == nil {
		return true, nil
	}
	return IsTestAssignedTo(ctx, testID, userID)
}

// SetTestItems replaces a test's items with the given set.
func SetTestItems(ctx context.Context, testID uuid.UUID, items []TestItemInput) error {
	crows, err := database.Query(ctx, "tests/items/clear", testID)
	if err != nil {
		return err
	}
	crows.Close()
	for i, it := range items {
		rows, err := database.Query(ctx, "tests/items/add", testID, it.ExerciseID, it.TrackReps, it.TrackWeight, it.TrackTime, it.TargetValue, i)
		if err != nil {
			return err
		}
		rows.Close()
	}
	return nil
}

// TestItems returns the stored items of a test (id, exercise, metric).
func TestItems(ctx context.Context, testID uuid.UUID) ([]TestItemRow, error) {
	var rows []TestItemRow
	if err := database.QuerySelect("tests/items/list", &rows, testID); err != nil {
		return nil, err
	}
	return rows, nil
}

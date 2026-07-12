package models

import (
	"context"
	"errors"
	"time"

	database "github.com/socious-io/pkg_database"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
)

var ErrTestRequestNotFound = errors.New("test request not found")

// TestRequest is an assessment to perform. Either a coach asks an athlete to take
// a test (test_id + coach_id set) or an athlete self-assesses (both NULL, own
// name). Lifecycle: PENDING -> SUBMITTED -> SEEN (coach acknowledges).
type TestRequest struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	TestID      *uuid.UUID `db:"test_id" json:"test_id,omitempty"`
	CoachID     *uuid.UUID `db:"coach_id" json:"coach_id,omitempty"`
	AthleteID   uuid.UUID  `db:"athlete_id" json:"athlete_id"`
	Name        *string    `db:"name" json:"name,omitempty"`
	Status      string     `db:"status" json:"status"`
	Note        *string    `db:"note" json:"note,omitempty"`
	SubmittedAt *time.Time `db:"submitted_at" json:"submitted_at,omitempty"`
	SeenAt      *time.Time `db:"seen_at" json:"seen_at,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
	// Hydrated in fetch.sql (no N+1). Athlete/Coach are full users; the rest are
	// passthrough JSON (test summary, items to perform, submitted records).
	Athlete     *User          `db:"-" json:"athlete"`
	AthleteJson types.JSONText `db:"athlete" json:"-"`
	Coach       *User          `db:"-" json:"coach"`
	CoachJson   types.JSONText `db:"coach" json:"-"`
	Test        types.JSONText `db:"test" json:"test"`
	Items       types.JSONText `db:"items" json:"items"`
	Records     types.JSONText `db:"records" json:"records"`
}

func (TestRequest) TableName() string  { return "test_requests" }
func (TestRequest) FetchQuery() string { return "test_requests/fetch" }

// SubmittedRecord is one result the athlete submits for a test item. An item may
// be compound (e.g. weighted pull-up), so it can carry reps + weight + time.
type SubmittedRecord struct {
	TestItemID uuid.UUID `json:"test_item_id" binding:"required"`
	Reps       *float64  `json:"reps"`
	Weight     *float64  `json:"weight"`
	Time       *float64  `json:"time"`
}

// SelfRecord is one exercise result in a self-assessment. The athlete picks the
// exercise directly (no test item) and records any of reps/weight/time.
type SelfRecord struct {
	ExerciseID uuid.UUID `json:"exercise_id" binding:"required"`
	Reps       *float64  `json:"reps"`
	Weight     *float64  `json:"weight"`
	Time       *float64  `json:"time"`
}

// CoachAssignment is one (protocol, client) the coach assigned, with the client
// and their run stats — the coach's mirror of the athlete's assigned list.
type CoachAssignment struct {
	TestID      uuid.UUID      `db:"test_id" json:"test_id"`
	AthleteID   uuid.UUID      `db:"athlete_id" json:"athlete_id"`
	AssignedAt  time.Time      `db:"assigned_at" json:"assigned_at"`
	TestName    string         `db:"test_name" json:"test_name"`
	ItemCount   int            `db:"item_count" json:"item_count"`
	RunsCount   int            `db:"runs_count" json:"runs_count"`
	LastRunAt   *time.Time     `db:"last_run_at" json:"last_run_at,omitempty"`
	Athlete     *User          `db:"-" json:"athlete"`
	AthleteJson types.JSONText `db:"athlete" json:"-"`
}

// ListCoachAssignments returns the protocols a coach assigned to clients, one row
// per (protocol, client), with the client and run stats.
func ListCoachAssignments(ctx context.Context, coachID uuid.UUID) ([]CoachAssignment, error) {
	items := []CoachAssignment{}
	if err := database.QuerySelect("tests/coach_assignments", &items, coachID); err != nil {
		return nil, err
	}
	if err := database.UnmarshalJSONTextFields(&items); err != nil {
		return nil, err
	}
	return items, nil
}

func CreateTestRequest(ctx context.Context, tr *TestRequest) error {
	rows, err := database.Query(ctx, "test_requests/create", tr.TestID, tr.CoachID, tr.AthleteID, tr.Note)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(tr); err != nil {
			return err
		}
	}
	return nil
}

func GetTestRequest(ctx context.Context, id uuid.UUID) (*TestRequest, error) {
	tr := new(TestRequest)
	if err := database.Fetch(tr, id); err != nil {
		return nil, ErrTestRequestNotFound
	}
	return tr, nil
}

func listTestRequests(query string, who uuid.UUID, status string, p database.Paginate) ([]TestRequest, int, error) {
	var (
		items     = []TestRequest{}
		fetchList []database.FetchList
		ids       []interface{}
		total     int
	)
	if err := database.QuerySelect(query, &fetchList, who, status, p.Limit, p.Offset); err != nil {
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

func ListCoachTestRequests(ctx context.Context, coachID uuid.UUID, status string, p database.Paginate) ([]TestRequest, int, error) {
	return listTestRequests("test_requests/list_by_coach", coachID, status, p)
}

func ListAthleteTestRequests(ctx context.Context, athleteID uuid.UUID, status string, p database.Paginate) ([]TestRequest, int, error) {
	return listTestRequests("test_requests/list_by_athlete", athleteID, status, p)
}

// SubmitTestRequest records the athlete's results as workout_logs and moves the
// request to SUBMITTED. Only the assigned athlete can submit a PENDING request.
// The status flip + all record inserts are one transaction, so a mid-way failure
// never leaves a SUBMITTED request with missing records.
func SubmitTestRequest(ctx context.Context, requestID, athleteID uuid.UUID, submitted []SubmittedRecord) error {
	tx, err := database.GetDB().BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	// Atomically flip PENDING -> SUBMITTED for the assigned athlete.
	tr := new(TestRequest)
	rows, err := database.TxQuery(ctx, tx, "test_requests/submit", requestID, athleteID)
	if err != nil {
		return err
	}
	found := false
	for rows.Next() {
		if err := rows.StructScan(tr); err != nil {
			rows.Close()
			return err
		}
		found = true
	}
	rows.Close()
	if !found {
		return errors.New("test request is not pending or not yours")
	}
	if tr.TestID == nil {
		return errors.New("self-assessments are submitted on creation")
	}

	items, err := TestItems(ctx, *tr.TestID)
	if err != nil {
		return err
	}
	byItem := make(map[uuid.UUID]TestItemRow, len(items))
	for _, it := range items {
		byItem[it.ID] = it
	}

	// Replace any prior records for this request, then insert the new ones.
	crows, err := database.TxQuery(ctx, tx, "test_requests/clear_records", requestID)
	if err != nil {
		return err
	}
	crows.Close()

	for _, s := range submitted {
		item, ok := byItem[s.TestItemID]
		if !ok {
			continue
		}
		// A compound item records only the metrics it tracks, in one row.
		var reps, weight, tm *float64
		if item.TrackReps {
			reps = s.Reps
		}
		if item.TrackWeight {
			weight = s.Weight
		}
		if item.TrackTime {
			tm = s.Time
		}
		if _, err := insertTestRecordTx(ctx, tx, requestID, item.ExerciseID, reps, weight, tm); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// RunProtocol records one run of an athlete's own protocol (a test they own):
// a SUBMITTED self-request linked to the test, plus its records mapped from the
// protocol's items. Atomic — a failed record insert rolls back the whole run.
func RunProtocol(ctx context.Context, testID, athleteID uuid.UUID, submitted []SubmittedRecord) (*TestRequest, error) {
	items, err := TestItems(ctx, testID)
	if err != nil {
		return nil, err
	}
	byItem := make(map[uuid.UUID]TestItemRow, len(items))
	for _, it := range items {
		byItem[it.ID] = it
	}

	tx, err := database.GetDB().BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	tr := new(TestRequest)
	rows, err := database.TxQuery(ctx, tx, "test_requests/create_run", testID, athleteID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		if err := rows.StructScan(tr); err != nil {
			rows.Close()
			return nil, err
		}
	}
	rows.Close()

	inserted := 0
	for _, s := range submitted {
		item, ok := byItem[s.TestItemID]
		if !ok {
			continue
		}
		var reps, weight, tm *float64
		if item.TrackReps {
			reps = s.Reps
		}
		if item.TrackWeight {
			weight = s.Weight
		}
		if item.TrackTime {
			tm = s.Time
		}
		ok, err := insertTestRecordTx(ctx, tx, tr.ID, item.ExerciseID, reps, weight, tm)
		if err != nil {
			return nil, err
		}
		if ok {
			inserted++
		}
	}
	if inserted == 0 {
		return nil, errors.New("a run needs at least one result")
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return tr, nil
}

// ListProtocolRuns returns an athlete's runs of a protocol, newest first.
func ListProtocolRunsPaginated(ctx context.Context, testID, athleteID uuid.UUID, p database.Paginate) ([]TestRequest, int, error) {
	items := []TestRequest{}
	var fetchList []database.FetchList
	if err := database.QuerySelect("test_requests/list_runs", &fetchList, testID, athleteID, p.Limit, p.Offset); err != nil {
		return nil, 0, err
	}
	if len(fetchList) == 0 {
		return items, 0, nil
	}
	total := fetchList[0].TotalCount
	ids := make([]interface{}, 0, len(fetchList))
	for _, f := range fetchList {
		ids = append(ids, f.ID)
	}
	if err := database.Fetch(&items, ids...); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// CreateSelfAssessment records an athlete's own assessment (no coach, no test
// template) as a SUBMITTED request plus its workout_logs rows, atomically — a
// failed record insert rolls back the request row instead of orphaning it.
func CreateSelfAssessment(ctx context.Context, athleteID uuid.UUID, name string, records []SelfRecord) (*TestRequest, error) {
	tx, err := database.GetDB().BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck // no-op once committed

	tr := new(TestRequest)
	rows, err := database.TxQuery(ctx, tx, "test_requests/create_self", athleteID, name)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		if err := rows.StructScan(tr); err != nil {
			rows.Close()
			return nil, err
		}
	}
	rows.Close()

	inserted := 0
	for _, r := range records {
		ok, err := insertTestRecordTx(ctx, tx, tr.ID, r.ExerciseID, r.Reps, r.Weight, r.Time)
		if err != nil {
			return nil, err
		}
		if ok {
			inserted++
		}
	}
	if inserted == 0 {
		return nil, errors.New("a self-assessment needs at least one result")
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return tr, nil
}

// insertTestRecordTx writes one workout_logs row for an assessment within a tx.
// Reps/time are stored as ints; it reports whether a row was actually written
// (false when the item had no values).
func insertTestRecordTx(ctx context.Context, tx *sqlx.Tx, requestID, exerciseID uuid.UUID, repsF, weight, timeF *float64) (bool, error) {
	var reps, duration *int
	if repsF != nil {
		v := int(*repsF)
		reps = &v
	}
	if timeF != nil {
		v := int(*timeF)
		duration = &v
	}
	if reps == nil && weight == nil && duration == nil {
		return false, nil
	}
	rows, err := database.TxQuery(ctx, tx, "test_requests/add_record", requestID, exerciseID, reps, weight, duration)
	if err != nil {
		return false, err
	}
	rows.Close()
	return true, nil
}

// MarkTestRequestSeen acknowledges a SUBMITTED assessment (the requesting coach,
// or any coach connected to the athlete for a self-assessment).
func MarkTestRequestSeen(ctx context.Context, requestID, coachID uuid.UUID) error {
	rows, err := database.Query(ctx, "test_requests/seen", requestID, coachID)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return errors.New("assessment is not awaiting review or not accessible")
	}
	return nil
}

package models

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"coachwise/src/database"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
)

type Exercise struct {
	ID          uuid.UUID         `json:"id" db:"id"`
	UserID      *uuid.UUID        `json:"user_id" db:"user_id"`
	Slug        *string           `json:"slug" db:"slug"`
	Name        string            `json:"name" db:"name"`
	Description string            `json:"description" db:"description"`
	// Localized name/description; the plain Name/Description are the fallback.
	NameI18n        LocalizedText `json:"name_i18n" db:"name_i18n"`
	DescriptionI18n LocalizedText `json:"description_i18n" db:"description_i18n"`
	Public          bool              `json:"public" db:"public"`
	SportType       ExerciseSportType `json:"sport_type" db:"sport_type"`
	CategoryID      *uuid.UUID        `json:"category_id" db:"category_id"`
	MediaID         *uuid.UUID        `json:"media_id" db:"media_id"`
	// Which extra actuals an athlete logs per set (reps/duration come from the
	// set prescription). Weight defaults on to match the historic logger.
	TrackWeight   bool `json:"track_weight" db:"track_weight"`
	TrackDistance bool `json:"track_distance" db:"track_distance"`
	TrackGrade    bool `json:"track_grade" db:"track_grade"`
	TrackHeight   bool `json:"track_height" db:"track_height"`
	// SINGLE (a movement) or GROUP (a circuit that references other exercises
	// and repeats them as rounds). Rounds/RoundRest/RoundDuration only apply to
	// a GROUP: RoundDuration set means run for that long instead of a count.
	Kind          string         `json:"kind" db:"kind"`
	Rounds        *int           `json:"rounds" db:"rounds"`
	RoundRest     time.Duration  `json:"round_rest" db:"round_rest"`
	RoundDuration *time.Duration `json:"round_duration" db:"round_duration"`
	Items         []ExerciseItem `json:"items" db:"-"`

	Media     *Media    `json:"media,omitempty" db:"-"`
	Sets      []Set     `json:"sets" db:"-"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	SetsJson  types.JSONText `db:"sets" json:"-"`
	MediaJson types.JSONText `db:"media" json:"-"`
	ItemsJson types.JSONText `db:"items" json:"-"`
	// Absorbs the generated tsvector column from `SELECT e.*`; not serialized.
	SearchVector *string `db:"search_vector" json:"-"`
	DeletedAt *time.Time `db:"deleted_at" json:"-"`
}

// ExerciseKind values.
const (
	ExerciseKindSingle = "SINGLE"
	ExerciseKindGroup  = "GROUP"
)

// ExerciseItem is one child of a GROUP exercise: which exercise to perform in a
// round, and its prescription for that round (one set, reps or time).
type ExerciseItem struct {
	ID         uuid.UUID      `json:"id" db:"id"`
	GroupID    uuid.UUID      `json:"group_id" db:"group_id"`
	ExerciseID uuid.UUID      `json:"exercise_id" db:"exercise_id"`
	ItemOrder  int            `json:"item_order" db:"item_order"`
	RepCount   *int           `json:"rep_count" db:"rep_count"`
	Duration   *time.Duration `json:"duration" db:"duration"`
	RestTime   time.Duration  `json:"rest_time" db:"rest_time"`
	CreatedAt  time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at" db:"updated_at"`
	// The referenced exercise, hydrated by fetch.sql so the runner has its name,
	// media and track flags without a second call. Always SINGLE (one level).
	Exercise *Exercise `json:"exercise,omitempty" db:"-"`
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

func (Exercise) TableName() string {
	return "exercises"
}

func (Exercise) FetchQuery() string {
	return "exercises/fetch"
}

// ListExercisesPaginated returns a page of exercises plus the total count.
// `search` is free text (any language); it's turned into a prefix tsquery matched
// against the exercises.search_vector (name/description + i18n + slug).
func ListExercisesPaginated(ctx context.Context, viewerID uuid.UUID, public *bool, search, category, sport string, p database.Paginate) ([]Exercise, int, error) {
	var (
		exercises = []Exercise{}
		fetchList []database.FetchList
		ids       []interface{}
		total     int
	)

	if err := database.QuerySelect("exercises/list", &fetchList, public, toTSQueryPrefix(search), p.Limit, p.Offset, category, sport, viewerID); err != nil {
		return nil, 0, err
	}
	if len(fetchList) == 0 {
		return exercises, 0, nil
	}
	total = fetchList[0].TotalCount
	for _, f := range fetchList {
		ids = append(ids, f.ID)
	}

	if err := database.Fetch(&exercises, ids...); err != nil {
		return nil, 0, err
	}
	// fetch.sql aggregates sets/media into JSON columns; hydrate them per row so
	// the list carries media (animations) and sets like the detail endpoint does.
	for i := range exercises {
		_ = exercises[i].SetsJson.Unmarshal(&exercises[i].Sets)
		_ = exercises[i].ItemsJson.Unmarshal(&exercises[i].Items)
		_ = exercises[i].MediaJson.Unmarshal(&exercises[i].Media)
	}

	return exercises, total, nil
}

// toTSQueryPrefix turns free-text search into a safe prefix tsquery string, e.g.
// "camp ladd" -> "camp:* & ladd:*". Only letters/digits survive tokenizing, so
// the result is always valid input for to_tsquery. Empty when there's no query.
func toTSQueryPrefix(search string) string {
	var tokens []string
	var b strings.Builder
	flush := func() {
		if b.Len() > 0 {
			tokens = append(tokens, b.String()+":*")
			b.Reset()
		}
	}
	for _, r := range strings.ToLower(strings.TrimSpace(search)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return strings.Join(tokens, " & ")
}

// writeItems rebuilds a GROUP's children inside the caller's transaction, the
// same clear-then-insert the sets use so order, additions and removals all land
// in one write. A SINGLE exercise has none, and clearing is harmless there.
func (e *Exercise) writeItems(ctx context.Context, tx *sqlx.Tx) error {
	rows, err := database.TxQuery(ctx, tx, "exercises/delete_items", e.ID)
	if err != nil {
		return err
	}
	// Drain it: an unread result leaves the connection mid-statement and the
	// next query on this tx dies with "unexpected Parse response".
	rows.Close()
	if e.Kind != ExerciseKindGroup || len(e.Items) == 0 {
		return nil
	}
	for i := range e.Items {
		e.Items[i].GroupID = e.ID
		e.Items[i].ItemOrder = i + 1
	}
	_, err = database.TxExecuteQuery(tx, "exercises/create_items", e.Items)
	return err
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
		e.UserID, e.Name, e.Description, e.Public, e.SportType, e.MediaID,
		e.TrackWeight, e.TrackDistance, e.TrackGrade, e.TrackHeight,
		e.Kind, e.Rounds, e.RoundRest, e.RoundDuration,
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

	if len(e.Sets) > 0 {
		if _, err := database.TxExecuteQuery(tx, "exercises/create_sets", e.Sets); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := e.writeItems(ctx, tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if err := database.Fetch(e, e.ID); err != nil {
		return err
	}
	_ = e.SetsJson.Unmarshal(&e.Sets)
	_ = e.ItemsJson.Unmarshal(&e.Items)
	_ = e.MediaJson.Unmarshal(&e.Media)
	return nil
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
		e.ID, e.Name, e.Description, e.Public, e.SportType, e.MediaID,
		e.TrackWeight, e.TrackDistance, e.TrackGrade, e.TrackHeight,
		e.Kind, e.Rounds, e.RoundRest, e.RoundDuration,
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

	// Rebuild sets to keep order and allow additions/removals
	if _, err := tx.ExecContext(ctx, "DELETE FROM sets WHERE exercise_id=$1", e.ID); err != nil {
		tx.Rollback()
		return err
	}
	for i := range e.Sets {
		e.Sets[i].ExerciseID = e.ID
		e.Sets[i].SetNumber = i + 1
	}
	if len(e.Sets) > 0 {
		if _, err := database.TxExecuteQuery(tx, "exercises/create_sets", e.Sets); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := e.writeItems(ctx, tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if err := database.Fetch(e, e.ID); err != nil {
		return err
	}
	_ = e.SetsJson.Unmarshal(&e.Sets)
	_ = e.ItemsJson.Unmarshal(&e.Items)
	_ = e.MediaJson.Unmarshal(&e.Media)
	return nil
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
	_ = e.SetsJson.Unmarshal(&e.Sets)
	_ = e.ItemsJson.Unmarshal(&e.Items)
	_ = e.MediaJson.Unmarshal(&e.Media)
	return e, nil
}

// ExerciseReachableViaPlan reports whether an exercise is part of any plan
// assigned to the user — so a client can open a coach's personal exercise that
// their assigned plan references.
func ExerciseReachableViaPlan(exerciseID, userID uuid.UUID) (bool, error) {
	var hits []int
	if err := database.QuerySelect("exercises/reachable_via_plan", &hits, exerciseID, userID); err != nil {
		return false, err
	}
	return len(hits) > 0, nil
}

func DeleteExercise(ctx context.Context, id uuid.UUID) error {
	rows, err := database.Query(ctx, "exercises/delete", id)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return fmt.Errorf("exercise not found")
	}
	return nil
}

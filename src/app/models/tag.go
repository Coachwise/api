package models

import (
	"context"
	"strings"
	"time"

	"coachwise/src/database"

	"github.com/google/uuid"
)

type Tag struct {
	ID             uuid.UUID `db:"id" json:"id"`
	Name           string    `db:"name" json:"name"`
	NormalizedName string    `db:"normalized_name" json:"normalized_name"`
	IsSystem       bool      `db:"is_system" json:"is_system"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}

func (Tag) TableName() string {
	return "tags"
}

func (Tag) FetchQuery() string {
	return "tags/fetch"
}

func NormalizeTagName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func (t *Tag) Create(ctx context.Context) error {
	t.NormalizedName = NormalizeTagName(t.Name)
	rows, err := database.Query(
		ctx,
		"tags/create",
		t.Name, t.NormalizedName, t.IsSystem,
	)
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

func GetOrCreateTag(ctx context.Context, name string, isSystem bool) (*Tag, error) {
	normalizedName := NormalizeTagName(name)

	tag := new(Tag)
	err := database.Get(tag, "tags/fetch_by_normalized_name", normalizedName)
	if err == nil {
		return tag, nil
	}

	tag.Name = name
	tag.NormalizedName = normalizedName
	tag.IsSystem = isSystem

	if err := tag.Create(ctx); err != nil {
		return nil, err
	}

	return tag, nil
}

func AddTagToWorkoutLog(ctx context.Context, workoutLogID, tagID uuid.UUID) error {
	_, err := database.Query(ctx, "workout_logs/add_tag", workoutLogID, tagID)
	return err
}

func AddTagToFeed(ctx context.Context, feedID, tagID uuid.UUID) error {
	_, err := database.Query(ctx, "feed/tags/add", feedID, tagID)
	return err
}

func GetWorkoutLogTags(ctx context.Context, workoutLogID uuid.UUID) ([]Tag, error) {
	var (
		tags      []Tag
		fetchList []database.FetchList
		ids       []interface{}
	)

	if err := database.QuerySelect("workout_logs/fetch_tags", &fetchList, workoutLogID); err != nil {
		return nil, err
	}

	if len(fetchList) < 1 {
		return tags, nil
	}

	for _, f := range fetchList {
		ids = append(ids, f.ID)
	}

	if err := database.Fetch(&tags, ids...); err != nil {
		return nil, err
	}

	return tags, nil
}

func GetFeedTags(ctx context.Context, feedID uuid.UUID) ([]Tag, error) {
	var (
		tags      []Tag
		fetchList []database.FetchList
		ids       []interface{}
	)

	if err := database.QuerySelect("feed/tags/fetch", &fetchList, feedID); err != nil {
		return nil, err
	}

	if len(fetchList) < 1 {
		return tags, nil
	}

	for _, f := range fetchList {
		ids = append(ids, f.ID)
	}

	if err := database.Fetch(&tags, ids...); err != nil {
		return nil, err
	}

	return tags, nil
}

func SearchTags(ctx context.Context, query string, sportType string, limit int, offset int) ([]Tag, int, error) {
	var (
		tags      []Tag
		fetchList []database.FetchList
		ids       []interface{}
	)

	if err := database.QuerySelect("tags/search", &fetchList, query, sportType, limit, offset); err != nil {
		return nil, 0, err
	}

	if len(fetchList) < 1 {
		return tags, 0, nil
	}

	for _, f := range fetchList {
		ids = append(ids, f.ID)
	}

	if err := database.Fetch(&tags, ids...); err != nil {
		return nil, 0, err
	}

	return tags, fetchList[0].TotalCount, nil
}

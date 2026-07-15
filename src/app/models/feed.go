package models

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"coachwise/src/database"

	"github.com/google/uuid"
)

type Feed struct {
	ID           uuid.UUID   `db:"id" json:"id"`
	UserID       uuid.UUID   `db:"user_id" json:"user_id"`
	Body         *string     `db:"body" json:"body,omitempty"`
	Location     *string     `db:"location" json:"location,omitempty"`
	Visibility   string      `db:"visibility" json:"visibility"`
	CreatedAt    time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time   `db:"updated_at" json:"updated_at"`
	Media        []FeedMedia `db:"-" json:"media"`
	Tags         []Tag       `db:"-" json:"tags"`
	LikeCount    int         `db:"like_count" json:"like_count"`
	CommentCount int         `db:"comment_count" json:"comment_count"`
	Liked        bool        `db:"liked" json:"liked"`
	DeletedAt *time.Time `db:"deleted_at" json:"-"`
}

type FeedMedia struct {
	ID           uuid.UUID `db:"id" json:"id"`
	FeedID       uuid.UUID `db:"feed_id" json:"feed_id"`
	Kind         string    `db:"kind" json:"kind"`
	URL          string    `db:"url" json:"url"`
	ThumbnailURL *string   `db:"thumbnail_url" json:"thumbnail_url,omitempty"`
	OrderIndex   int       `db:"order_index" json:"order_index"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type FeedComment struct {
	ID        uuid.UUID `db:"id" json:"id"`
	FeedID    uuid.UUID `db:"feed_id" json:"feed_id"`
	UserID    uuid.UUID `db:"user_id" json:"user_id"`
	Body      string    `db:"body" json:"body"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type feedListRow struct {
	ID           uuid.UUID `db:"id"`
	TotalCount   int       `db:"total_count"`
	LikeCount    int       `db:"like_count"`
	CommentCount int       `db:"comment_count"`
	Liked        bool      `db:"liked"`
}

func (Feed) TableName() string {
	return "feeds"
}

func (Feed) FetchQuery() string {
	return "feed/fetch"
}

func (f *Feed) Create(ctx context.Context, media []FeedMedia, tagNames []string) error {
	if f.Visibility == "" {
		f.Visibility = "PUBLIC"
	}

	tx, err := database.GetDB().Beginx()
	if err != nil {
		return err
	}

	rows, err := database.TxQuery(ctx, tx, "feed/create", f.UserID, f.Body, f.Location, f.Visibility)
	if err != nil {
		tx.Rollback()
		return err
	}
	for rows.Next() {
		if err := rows.StructScan(f); err != nil {
			rows.Close()
			tx.Rollback()
			return err
		}
	}
	rows.Close()

	for i := range media {
		media[i].FeedID = f.ID
		media[i].Kind = strings.ToUpper(media[i].Kind)
		mRows, err := database.TxQuery(
			ctx,
			tx,
			"feed/media/create",
			media[i].FeedID,
			media[i].Kind,
			media[i].URL,
			media[i].ThumbnailURL,
			media[i].OrderIndex,
		)
		if err != nil {
			tx.Rollback()
			return err
		}
		for mRows.Next() {
			var item FeedMedia
			if err := mRows.StructScan(&item); err != nil {
				mRows.Close()
				tx.Rollback()
				return err
			}
			f.Media = append(f.Media, item)
		}
		mRows.Close()
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if len(tagNames) > 0 {
		for _, tagName := range tagNames {
			if strings.TrimSpace(tagName) == "" {
				continue
			}
			tag, err := GetOrCreateTag(ctx, tagName, false)
			if err != nil {
				continue
			}
			_ = AddTagToFeed(ctx, f.ID, tag.ID)
		}
	}

	if err := f.loadMediaAndTags(ctx); err != nil {
		return err
	}

	return nil
}

func GetFeed(id uuid.UUID) (*Feed, error) {
	f := new(Feed)
	if err := database.Fetch(f, id); err != nil {
		return nil, err
	}
	return f, nil
}

func GetFeedWithDetails(ctx context.Context, id uuid.UUID, viewerID uuid.UUID) (*Feed, error) {
	rows, err := database.Query(ctx, "feed/fetch_with_counts", id, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, sql.ErrNoRows
	}

	feed := new(Feed)
	if err := rows.StructScan(feed); err != nil {
		return nil, err
	}

	if err := feed.loadMediaAndTags(ctx); err != nil {
		return nil, err
	}

	return feed, nil
}

func ListFeeds(ctx context.Context, viewerID uuid.UUID, limit int, offset int) ([]Feed, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var rows []feedListRow
	if err := database.QuerySelect("feed/list", &rows, viewerID, limit, offset); err != nil {
		return nil, 0, err
	}

	if len(rows) == 0 {
		return []Feed{}, 0, nil
	}

	var (
		ids      []interface{}
		rowIndex = make(map[uuid.UUID]feedListRow)
	)
	for _, r := range rows {
		ids = append(ids, r.ID)
		rowIndex[r.ID] = r
	}

	var feeds []Feed
	if err := database.Fetch(&feeds, ids...); err != nil {
		return nil, 0, err
	}

	for i := range feeds {
		if data, ok := rowIndex[feeds[i].ID]; ok {
			feeds[i].LikeCount = data.LikeCount
			feeds[i].CommentCount = data.CommentCount
			feeds[i].Liked = data.Liked
		}
		if err := feeds[i].loadMediaAndTags(ctx); err != nil {
			return nil, 0, err
		}
	}

	return feeds, rows[0].TotalCount, nil
}

func (f *Feed) loadMediaAndTags(ctx context.Context) error {
	media, err := GetFeedMedia(ctx, f.ID)
	if err != nil {
		return err
	}
	f.Media = media

	tags, err := GetFeedTags(ctx, f.ID)
	if err != nil {
		return err
	}
	f.Tags = tags
	return nil
}

func DeleteFeed(ctx context.Context, feedID, userID uuid.UUID) error {
	rows, err := database.Query(ctx, "feed/delete", feedID, userID)
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		return nil
	}
	return errors.New("feed not found or access denied")
}

func GetFeedMedia(ctx context.Context, feedID uuid.UUID) ([]FeedMedia, error) {
	rows, err := database.Query(ctx, "feed/media/list_by_feed", feedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	media := []FeedMedia{}
	for rows.Next() {
		var item FeedMedia
		if err := rows.StructScan(&item); err != nil {
			return nil, err
		}
		media = append(media, item)
	}
	return media, nil
}

func CreateFeedComment(ctx context.Context, comment *FeedComment) error {
	rows, err := database.Query(ctx, "feed/comments/create", comment.FeedID, comment.UserID, comment.Body)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(comment); err != nil {
			return err
		}
	}
	return nil
}

func (fc *FeedComment) Create(ctx context.Context) error {
	return CreateFeedComment(ctx, fc)
}

func ListFeedComments(ctx context.Context, feedID uuid.UUID) ([]FeedComment, error) {
	rows, err := database.Query(ctx, "feed/comments/list_by_feed", feedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []FeedComment{}
	for rows.Next() {
		var c FeedComment
		if err := rows.StructScan(&c); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, nil
}

func CountFeedComments(ctx context.Context, feedID uuid.UUID) (int, error) {
	var count int
	if err := database.Get(&count, "feed/comments/count", feedID); err != nil {
		return 0, err
	}
	return count, nil
}

func AddFeedLike(ctx context.Context, feedID, userID uuid.UUID) error {
	rows, err := database.Query(ctx, "feed/likes/create", feedID, userID)
	if err != nil {
		return err
	}
	defer rows.Close()
	return nil
}

func RemoveFeedLike(ctx context.Context, feedID, userID uuid.UUID) error {
	rows, err := database.Query(ctx, "feed/likes/delete", feedID, userID)
	if err != nil {
		return err
	}
	defer rows.Close()
	return nil
}

func CountFeedLikes(ctx context.Context, feedID uuid.UUID) (int, error) {
	var count int
	if err := database.Get(&count, "feed/likes/count", feedID); err != nil {
		return 0, err
	}
	return count, nil
}

func HasUserLikedFeed(ctx context.Context, feedID, userID uuid.UUID) (bool, error) {
	rows, err := database.Query(ctx, "feed/likes/check", feedID, userID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	if rows.Next() {
		var liked bool
		if err := rows.Scan(&liked); err != nil {
			return false, err
		}
		return liked, nil
	}
	return false, nil
}

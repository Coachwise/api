package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	database "github.com/socious-io/pkg_database"
)

type Media struct {
	ID        uuid.UUID `db:"id" json:"id"`
	UserID    uuid.UUID `db:"user_id" json:"-"`
	URL       string    `db:"url" json:"url"`
	Filename  string    `db:"filename" json:"filename"`
	SizeBytes int64     `db:"size_bytes" json:"size_bytes"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func (*Media) TableName() string {
	return "media"
}

func (*Media) FetchQuery() string {
	return "media/fetch"
}

func (m *Media) Scan(rows *sqlx.Rows) error {
	return rows.StructScan(m)
}

func CreateMedia(ctx context.Context, userID uuid.UUID, url string, filename string, sizeBytes int64) (*Media, error) {
	if sizeBytes < 0 {
		sizeBytes = 0
	}
	media := &Media{UserID: userID, URL: url, Filename: filename, SizeBytes: sizeBytes}
	rows, err := database.Query(ctx, "media/create", userID, url, filename, sizeBytes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(media); err != nil {
			return nil, err
		}
	}
	return media, nil
}

func ListMedia(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Media, error) {
	var items []Media
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	if err := database.QuerySelect("media/list_by_user", &items, userID, limit, offset); err != nil {
		return nil, err
	}
	return items, nil
}

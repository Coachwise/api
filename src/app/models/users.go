package models

import (
	"context"
	"strings"
	"time"

	"github.com/jmoiron/sqlx/types"
	database "github.com/socious-io/pkg_database"

	"github.com/google/uuid"
)

type User struct {
	ID              uuid.UUID `db:"id" json:"id"`
	Username        string    `db:"username" json:"username"`
	Email           string    `db:"email" json:"email"`
	Password        *string   `db:"password" json:"-"`
	JobTitle        *string   `db:"job_title" json:"job_title"`
	Bio             *string   `db:"bio" json:"bio"`
	FirstName       *string   `db:"first_name" json:"first_name"`
	LastName        *string   `db:"last_name" json:"last_name"`
	Phone           *string   `db:"phone" json:"phone"`
	Website         *string   `db:"website" json:"website"`
	Instagram       *string   `db:"instagram" json:"instagram"`
	Birthday        *time.Time `db:"birthday" json:"birthday"`
	Status          string    `db:"status" json:"status"`
	PasswordExpired bool      `db:"password_expired" json:"password_expired"`

	ProUntil *time.Time `db:"pro_until" json:"pro_until,omitempty"`
	Pro      bool       `db:"pro" json:"pro"`

	IsCoach bool `db:"is_coach" json:"is_coach"`

	// Computed per-viewer in the connection endpoints (not from a column).
	ConnectionStatus string `db:"-" json:"connection_status,omitempty"` // none|pending_outgoing|pending_incoming|connected
	IsConnected      bool   `db:"-" json:"is_connected"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`

	AvatarID   *uuid.UUID     `db:"avatar_id" json:"avatar_id"`
	Avatar     *Media         `db:"-" json:"avatar"`
	AvatarJson types.JSONText `db:"avatar" json:"-"`
	// Absorbs the generated tsvector from `SELECT u.*`; not serialized.
	SearchVector *string `db:"search_vector" json:"-"`
}

func (User) TableName() string {
	return "users"
}

func (User) FetchQuery() string {
	return "users/fetch"
}

func (u *User) Create(ctx context.Context) error {
	rows, err := database.Query(
		ctx,
		"users/register",
		u.FirstName, u.LastName, u.Username, u.Email, u.Password,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(u); err != nil {
			return err
		}
	}
	return database.Fetch(u, u.ID)
}

func (u *User) Verify(ctx context.Context) error {
	rows, err := database.Query(
		ctx,
		"users/verify",
		u.ID, u.Status,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(u); err != nil {
			return err
		}
	}
	return database.Fetch(u, u.ID)
}

func (u *User) ExpirePassword(ctx context.Context) error {
	rows, err := database.Query(
		ctx,
		"users/expire_password",
		u.ID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(u); err != nil {
			return err
		}
	}
	return database.Fetch(u, u.ID)
}

func (u *User) UpdatePassword(ctx context.Context) error {
	rows, err := database.Query(
		ctx,
		"users/update_password",
		u.ID, u.Password,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(u); err != nil {
			return err
		}
	}
	return database.Fetch(u, u.ID)
}

func (u *User) Update(ctx context.Context) error {
	rows, err := database.Query(
		ctx,
		"users/update",
		u.ID, u.FirstName, u.LastName, u.Bio, u.JobTitle, u.Phone, u.Username, u.AvatarID,
		u.Website, u.Instagram, u.Birthday,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(u); err != nil {
			return err
		}
	}
	return database.Fetch(u, u.ID)
}

func GetUser(id uuid.UUID) (*User, error) {
	u := new(User)
	if err := database.Fetch(u, id.String()); err != nil {
		return nil, err
	}
	return u, nil
}

func GetUserByEmail(email string) (*User, error) {
	u := new(User)
	if err := database.Get(u, "users/fetch_by_email", email); err != nil {
		return nil, err
	}
	database.Fetch(u, u.ID)
	return u, nil
}

func GetUserByUsername(username string) (*User, error) {
	u := new(User)
	if err := database.Get(u, "users/fetch_by_username", username); err != nil {
		return nil, err
	}
	database.Fetch(u, u.ID)
	return u, nil
}

func GetUserByPhone(phone string) (*User, error) {
	u := new(User)
	if err := database.Get(u, "users/fetch_by_phone", phone); err != nil {
		return nil, err
	}
	database.Fetch(u, u.ID)
	return u, nil
}

// GetOrCreatePhoneUser returns the account for a phone, creating a passwordless
// one (placeholder email, auto username, INACTIVE) on first use.
func GetOrCreatePhoneUser(ctx context.Context, phone string) (*User, error) {
	if u, err := GetUserByPhone(phone); err == nil {
		return u, nil
	}
	username := "u" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	email := phone + "@phone.coachwise.local"
	u := new(User)
	rows, err := database.Query(ctx, "users/create_phone", username, email, phone)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		if err := rows.StructScan(u); err != nil {
			rows.Close()
			return nil, err
		}
	}
	rows.Close()
	return GetUserByPhone(phone)
}

// ListUsersPaginated searches users by username or full name. When coachOnly is
// true it returns only users flagged as coaches; excludeID drops that user from
// the results (used to hide the requester from their own search).
func ListUsersPaginated(ctx context.Context, search string, coachOnly bool, excludeID uuid.UUID, p database.Paginate) ([]User, int, error) {
	var (
		items     = []User{}
		fetchList []database.FetchList
		ids       []interface{}
		total     int
	)

	if err := database.QuerySelect("users/list", &fetchList, toTSQueryPrefix(search), coachOnly, excludeID, p.Limit, p.Offset); err != nil {
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

func DeleteUser(ctx context.Context, id uuid.UUID) error {
	rows, err := database.Query(ctx, "users/delete", id)
	if err != nil {
		return err
	}
	rows.Close()
	return nil
}

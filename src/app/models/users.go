package models

import (
	"coachwise/src/database"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type User struct {
	ID        uuid.UUID `db:"id" json:"id"`
	Username  string    `db:"username" json:"username"`
	Email     string    `db:"email" json:"email"`
	Password  *string   `db:"password" json:"-"`
	JobTitle  *string   `db:"job_title" json:"job_title"`
	Bio       *string   `db:"bio" json:"-"`
	FirstName *string   `db:"first_name" json:"first_name"`
	LastName  *string   `db:"last_name" json:"last_name"`
	Phone     *string   `db:"phone" json:"phone"`
	AvatarID  uuid.UUID `db:"avatar_id" json:"avatar_id"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

func (*User) TableName() string {
	return "users"
}

func (*User) FetchQuery() string {
	return "users/fetch"
}

func (u *User) Scan(rows *sqlx.Rows) error {
	return rows.StructScan(u)
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
		if err := u.Scan(rows); err != nil {
			return err
		}
	}
	return nil
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
	return u, nil
}

package models

import (
	"context"
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
	Bio             *string   `db:"bio" json:"-"`
	FirstName       *string   `db:"first_name" json:"first_name"`
	LastName        *string   `db:"last_name" json:"last_name"`
	Phone           *string   `db:"phone" json:"phone"`
	Status          string    `db:"status" json:"status"`
	PasswordExpired bool      `db:"password_expired" json:"password_expired"`

	ProUntil *time.Time `db:"pro_until" json:"pro_until,omitempty"`
	Pro      bool       `db:"pro" json:"pro"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`

	AvatarID   *uuid.UUID     `db:"avatar_id" json:"avatar_id"`
	Avatar     *Media         `db:"-" json:"avatar"`
	AvatarJson types.JSONText `db:"avatar" json:"-"`
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

func ListUsers(ctx context.Context, username string) ([]User, error) {
	var users []User
	db := database.GetDB()
	query := "SELECT * FROM users"
	args := []interface{}{}
	if username != "" {
		query += " WHERE LOWER(username) LIKE LOWER($1)"
		args = append(args, "%"+username+"%")
	}
	if err := db.SelectContext(ctx, &users, query, args...); err != nil {
		return nil, err
	}
	return users, nil
}

func DeleteUser(ctx context.Context, id uuid.UUID) error {
	db := database.GetDB()
	if _, err := db.ExecContext(ctx, "DELETE FROM users WHERE id=$1", id); err != nil {
		return err
	}
	return nil
}

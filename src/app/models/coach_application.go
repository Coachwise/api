package models

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	database "github.com/socious-io/pkg_database"

	"github.com/google/uuid"
)

type CoachApplication struct {
	ID              uuid.UUID `db:"id" json:"id"`
	UserID          uuid.UUID `db:"user_id" json:"user_id"`
	FullName        string    `db:"full_name" json:"full_name"`
	Specialty       string    `db:"specialty" json:"specialty"`
	ExperienceYears int       `db:"experience_years" json:"experience_years"`
	Certifications  string    `db:"certifications" json:"certifications"`
	Bio             *string   `db:"bio" json:"bio,omitempty"`
	Website         *string   `db:"website" json:"website,omitempty"`
	Instagram       *string   `db:"instagram" json:"instagram,omitempty"`
	Status          string    `db:"status" json:"status"`
	// DecisionToken is the secret behind the approve/reject capability links;
	// never serialized to clients.
	DecisionToken string    `db:"decision_token" json:"-"`
	ReviewNote    *string   `db:"review_note" json:"review_note,omitempty"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

func generateDecisionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (a *CoachApplication) Create(ctx context.Context) error {
	token, err := generateDecisionToken()
	if err != nil {
		return err
	}
	a.DecisionToken = token
	rows, err := database.Query(ctx, "coach_applications/create",
		a.UserID, a.FullName, a.Specialty, a.ExperienceYears, a.Certifications,
		a.Bio, a.Website, a.Instagram, a.DecisionToken)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(a); err != nil {
			return err
		}
	}
	return nil
}

func GetCoachApplication(id uuid.UUID) (*CoachApplication, error) {
	app := new(CoachApplication)
	if err := database.Get(app, "coach_applications/fetch", id); err != nil {
		return nil, err
	}
	return app, nil
}

// LatestCoachApplication returns the user's most recent application (or an error
// when they have none).
func LatestCoachApplication(userID uuid.UUID) (*CoachApplication, error) {
	app := new(CoachApplication)
	if err := database.Get(app, "coach_applications/latest_by_user", userID); err != nil {
		return nil, err
	}
	return app, nil
}

// DecideCoachApplication sets a PENDING application to APPROVED/REJECTED when the
// decision token matches. On approval it makes the user a coach (the coaches
// insert fires the is_coach trigger).
func DecideCoachApplication(ctx context.Context, id uuid.UUID, token, status string, note *string) (*CoachApplication, error) {
	app := new(CoachApplication)
	found := false
	rows, err := database.Query(ctx, "coach_applications/set_status", id, token, status, note)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		if err := rows.StructScan(app); err != nil {
			rows.Close()
			return nil, err
		}
		found = true
	}
	rows.Close()
	if !found {
		return nil, errors.New("application not found, already decided, or invalid token")
	}

	if status == "APPROVED" {
		crows, err := database.Query(ctx, "coaches/create", app.UserID, specialtyToSport(app.Specialty))
		if err != nil {
			return nil, err
		}
		crows.Close()
	}
	return app, nil
}

// specialtyToSport maps a free-form application specialty onto the sports enum
// used by the coaches table.
func specialtyToSport(specialty string) string {
	if strings.Contains(strings.ToLower(specialty), "climb") {
		return "CLIMBING"
	}
	return "FITNESS"
}

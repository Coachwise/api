package models

import (
	"context"
	"errors"
	"time"

	database "github.com/socious-io/pkg_database"

	"github.com/google/uuid"
)

// ErrClientHasPackage is returned when enrolling a client who already has a
// different active package from the same coach (one package per client).
var ErrClientHasPackage = errors.New("client already has a package; remove it before assigning another")

// PackageSubscription is the client relationship: a user enrolled in a coach's
// package (coach-assigned or athlete-subscribed). No payment yet.
type PackageSubscription struct {
	ID        uuid.UUID `db:"id" json:"id"`
	PackageID uuid.UUID `db:"package_id" json:"package_id"`
	CoachID   uuid.UUID `db:"coach_id" json:"coach_id"`
	ClientID  uuid.UUID `db:"client_id" json:"client_id"`
	Status    string    `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`

	TotalCount int `db:"total_count" json:"-"` // window count for pagination
}

// UnsubscribeClient removes a client's subscription to a package and unassigns
// the plans that came from that package. Manually-assigned plans (package_id
// NULL) are left untouched — those must be unassigned individually.
func UnsubscribeClient(ctx context.Context, packageID, clientID uuid.UUID) error {
	rows, err := database.Query(ctx, "plans/assignees/delete_by_package", clientID, packageID)
	if err != nil {
		return err
	}
	rows.Close()
	srows, err := database.Query(ctx, "subscriptions/delete", packageID, clientID)
	if err != nil {
		return err
	}
	srows.Close()
	return nil
}

// ListClientSubscriptions returns the user's active package subscriptions.
func ListClientSubscriptionsPaginated(ctx context.Context, clientID uuid.UUID, p database.Paginate) ([]PackageSubscription, int, error) {
	subs := []PackageSubscription{}
	if err := database.QuerySelect("subscriptions/by_client", &subs, clientID, p.Limit, p.Offset); err != nil {
		return nil, 0, err
	}
	total := 0
	if len(subs) > 0 {
		total = subs[0].TotalCount
	}
	return subs, total, nil
}

// ActiveSubscriptionFor returns the client's active subscription with the coach,
// or nil when they have none.
func ActiveSubscriptionFor(ctx context.Context, coachID, clientID uuid.UUID) *PackageSubscription {
	sub := new(PackageSubscription)
	if err := database.Get(sub, "subscriptions/active_for_client", coachID, clientID); err != nil {
		return nil
	}
	return sub
}

// EnrollClient enrolls a client in a package (idempotent) and assigns every plan
// bundled in the package to them. coachID is the package owner. A client may hold
// only one package per coach — enrolling in a different one is rejected.
func EnrollClient(ctx context.Context, packageID, coachID, clientID uuid.UUID) (*PackageSubscription, error) {
	if existing := ActiveSubscriptionFor(ctx, coachID, clientID); existing != nil && existing.PackageID != packageID {
		return nil, ErrClientHasPackage
	}
	sub := new(PackageSubscription)
	rows, err := database.Query(ctx, "subscriptions/enroll", packageID, coachID, clientID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		if err := rows.StructScan(sub); err != nil {
			rows.Close()
			return nil, err
		}
	}
	rows.Close()

	if err := AssignPackage(ctx, packageID, clientID, coachID); err != nil {
		return nil, err
	}
	return sub, nil
}

package models

import (
	"context"
	"errors"
	"time"

	"coachwise/src/database"

	"github.com/google/uuid"
)

// ErrClientHasPackage is returned when enrolling a client who already has a
// different active package from the same coach (one package per client).
var ErrClientHasPackage = errors.New("client already has a package; remove it before assigning another")

// ErrSubscriptionNotFound means there was no active subscription to cancel.
var ErrSubscriptionNotFound = errors.New("subscription not found")

// PackageSubscription is the client relationship: a user enrolled in a coach's
// package (coach-assigned or athlete-subscribed). No payment yet.
type PackageSubscription struct {
	ID        uuid.UUID `db:"id" json:"id"`
	PackageID uuid.UUID `db:"package_id" json:"package_id"`
	CoachID   uuid.UUID `db:"coach_id" json:"coach_id"`
	ClientID  uuid.UUID `db:"client_id" json:"client_id"`
	Status    string    `db:"status" json:"status"`
	// EndsAt is the term the client paid for. A refund is the unused part of it.
	EndsAt  *time.Time `db:"ends_at" json:"ends_at,omitempty"`
	OrderID *uuid.UUID `db:"order_id" json:"order_id,omitempty"`
	// Who ended it, when, and why — the record a refund or a dispute rests on.
	CanceledAt   *time.Time `db:"canceled_at" json:"canceled_at,omitempty"`
	CanceledBy   *uuid.UUID `db:"canceled_by" json:"canceled_by,omitempty"`
	CancelReason *string    `db:"cancel_reason" json:"cancel_reason,omitempty"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at" json:"-"`

	TotalCount int `db:"total_count" json:"-"` // window count for pagination
}

// CancelSubscription ends a client's subscription to a package and unassigns the
// plans that came from it. Manually-assigned plans (package_id NULL) are left
// untouched — those must be unassigned individually.
//
// The subscription row survives: it is soft-deleted and records who ended it and
// why. The refund is worked out from it (how much of the term was left), and if
// anyone argues about it later, it is the evidence. It also carries the term and
// the order, so it knows exactly what money it came from.
func CancelSubscription(ctx context.Context, packageID, clientID, canceledBy uuid.UUID, reason string) (*PackageSubscription, error) {
	rows, err := database.Query(ctx, "plans/assignees/delete_by_package", clientID, packageID)
	if err != nil {
		return nil, err
	}
	rows.Close()

	sub := new(PackageSubscription)
	srows, err := database.Query(ctx, "subscriptions/cancel", packageID, clientID, canceledBy, reason)
	if err != nil {
		return nil, err
	}
	defer srows.Close()
	if !srows.Next() {
		return nil, ErrSubscriptionNotFound
	}
	if err := srows.StructScan(sub); err != nil {
		return nil, err
	}
	return sub, nil
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

// IsCoachClient reports whether the client is actively enrolled in one of the
// coach's packages — the gate for a coach viewing that client's analytics.
func IsCoachClient(ctx context.Context, coachID, clientID uuid.UUID) (bool, error) {
	var hits []int
	if err := database.QuerySelect("subscriptions/is_client", &hits, coachID, clientID); err != nil {
		return false, err
	}
	return len(hits) > 0, nil
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
//
// endsAt is the term the client is entitled to, and orderID the money it came
// from (nil when a coach assigns a package by hand, with no purchase). A client
// who was previously cancelled is revived into the same row.
func EnrollClient(ctx context.Context, packageID, coachID, clientID uuid.UUID, endsAt time.Time, orderID *uuid.UUID) (*PackageSubscription, error) {
	if existing := ActiveSubscriptionFor(ctx, coachID, clientID); existing != nil && existing.PackageID != packageID {
		return nil, ErrClientHasPackage
	}
	sub := new(PackageSubscription)
	rows, err := database.Query(ctx, "subscriptions/enroll", packageID, coachID, clientID, endsAt, orderID)
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

// DefaultTermFrom is the term a subscription gets when no purchase set one — a
// coach handing a package to a client directly. One month, renewable by doing it
// again.
func DefaultTermFrom(t time.Time) time.Time {
	return t.UTC().AddDate(0, 1, 0)
}

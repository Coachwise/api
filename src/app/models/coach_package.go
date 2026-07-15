package models

import (
	"context"
	"errors"
	"time"

	"coachwise/src/database"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx/types"
)

var (
	ErrPackageNotFound     = errors.New("package not found")
	ErrPackageAccessDenied = errors.New("package not accessible")
)

// CoachPackage is a coach-owned, sellable "tier" that bundles a set of plans.
// Pricing/feature fields are metadata only for now (no payment processing).
type CoachPackage struct {
	ID               uuid.UUID      `db:"id" json:"id"`
	CoachID          uuid.UUID      `db:"coach_id" json:"coach_id"`
	Name             string         `db:"name" json:"name"`
	Description      *string        `db:"description" json:"description,omitempty"`
	BillingType      string         `db:"billing_type" json:"billing_type"` // SUBSCRIPTION | ONE_TIME
	Currency         string         `db:"currency" json:"currency"`
	PriceMonthly     *int           `db:"price_monthly" json:"price_monthly,omitempty"`
	PriceAnnual      *int           `db:"price_annual" json:"price_annual,omitempty"`
	PriceOneTime     *int           `db:"price_one_time" json:"price_one_time,omitempty"`
	TrialDays        int            `db:"trial_days" json:"trial_days"`
	CheckInFrequency *string        `db:"check_in_frequency" json:"check_in_frequency,omitempty"`
	VideoAccess      bool           `db:"video_access" json:"video_access"`
	NutritionGuides  bool           `db:"nutrition_guides" json:"nutrition_guides"`
	CustomFeatures   types.JSONText `db:"custom_features" json:"custom_features"`
	IsActive         bool           `db:"is_active" json:"is_active"`
	Popular          bool           `db:"popular" json:"popular"`
	CreatedAt        time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time      `db:"updated_at" json:"updated_at"`
	// Hydrated in fetch.sql (no N+1).
	PlanCount int            `db:"plan_count" json:"plan_count"`
	Plans     types.JSONText `db:"plans" json:"plans"`
	DeletedAt *time.Time `db:"deleted_at" json:"-"`
}

func (CoachPackage) TableName() string {
	return "coach_packages"
}

func (CoachPackage) FetchQuery() string {
	return "packages/fetch"
}

func (p *CoachPackage) Create(ctx context.Context) error {
	rows, err := database.Query(
		ctx,
		"packages/create",
		p.CoachID, p.Name, p.Description,
		p.PriceMonthly, p.PriceAnnual, p.PriceOneTime,
		p.TrialDays, p.CheckInFrequency, p.VideoAccess, p.NutritionGuides,
		p.CustomFeatures, p.IsActive, p.Popular,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(p); err != nil {
			return err
		}
	}
	return nil
}

func (p *CoachPackage) Update(ctx context.Context) error {
	rows, err := database.Query(
		ctx,
		"packages/update",
		p.ID, p.Name, p.Description,
		p.PriceMonthly, p.PriceAnnual, p.PriceOneTime,
		p.TrialDays, p.CheckInFrequency, p.VideoAccess, p.NutritionGuides,
		p.CustomFeatures, p.IsActive, p.Popular,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(p); err != nil {
			return err
		}
	}
	return nil
}

// GetPackageForCoach returns a package only when it belongs to the given coach.
func GetPackageForCoach(ctx context.Context, packageID, coachID uuid.UUID) (*CoachPackage, error) {
	p := new(CoachPackage)
	if err := database.Fetch(p, packageID); err != nil {
		return nil, ErrPackageNotFound
	}
	if p.CoachID != coachID {
		return nil, ErrPackageAccessDenied
	}
	return p, nil
}

func GetPackage(ctx context.Context, packageID uuid.UUID) (*CoachPackage, error) {
	p := new(CoachPackage)
	if err := database.Fetch(p, packageID); err != nil {
		return nil, ErrPackageNotFound
	}
	return p, nil
}

func DeletePackage(ctx context.Context, id uuid.UUID) error {
	rows, err := database.Query(ctx, "packages/delete", id)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return ErrPackageNotFound
	}
	return nil
}

func ListCoachPackagesPaginated(ctx context.Context, coachID uuid.UUID, p database.Paginate) ([]CoachPackage, int, error) {
	var (
		items     = []CoachPackage{}
		fetchList []database.FetchList
		ids       []interface{}
		total     int
	)

	if err := database.QuerySelect("packages/list", &fetchList, coachID, p.Limit, p.Offset); err != nil {
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

// ListCoachPackagesPublic returns a coach's active packages (athlete-facing).
func ListCoachPackagesPublic(ctx context.Context, coachID uuid.UUID) ([]CoachPackage, error) {
	var (
		items     = []CoachPackage{}
		fetchList []database.FetchList
		ids       []interface{}
	)
	if err := database.QuerySelect("packages/list_by_coach_public", &fetchList, coachID); err != nil {
		return nil, err
	}
	if len(fetchList) < 1 {
		return items, nil
	}
	for _, f := range fetchList {
		ids = append(ids, f.ID)
	}
	if err := database.Fetch(&items, ids...); err != nil {
		return nil, err
	}
	return items, nil
}

// SetPackagePlans replaces a package's bundled plans with the given set.
func SetPackagePlans(ctx context.Context, packageID uuid.UUID, planIDs []uuid.UUID) error {
	crows, err := database.Query(ctx, "packages/plans/clear", packageID)
	if err != nil {
		return err
	}
	crows.Close()
	for _, planID := range planIDs {
		rows, err := database.Query(ctx, "packages/plans/add", packageID, planID)
		if err != nil {
			return err
		}
		rows.Close()
	}
	return nil
}

// PackagePlanIDs returns the plan ids bundled into a package.
func PackagePlanIDs(ctx context.Context, packageID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	if err := database.QuerySelect("packages/plans/list_ids", &ids, packageID); err != nil {
		return nil, err
	}
	return ids, nil
}

// AssignPackage assigns every plan bundled in the package to the user, using the
// existing plan_assignees machinery (assigner = the coach).
func AssignPackage(ctx context.Context, packageID, userID, assignerID uuid.UUID) error {
	planIDs, err := PackagePlanIDs(ctx, packageID)
	if err != nil {
		return err
	}
	for _, planID := range planIDs {
		pkgID := packageID
		assignee := &PlanAssignee{
			PlanID:    planID,
			UserID:    userID,
			Assigner:  assignerID,
			PackageID: &pkgID, // stamp origin so unsubscribe can remove exactly these
		}
		if err := AssignPlan(ctx, assignee); err != nil {
			return err
		}
	}
	return nil
}

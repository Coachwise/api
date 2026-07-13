package models

import (
	"context"
	"time"

	"coachwise/src/database"

	"github.com/google/uuid"
)

// AssignedPlanInfo is a single plan a coach has assigned to one of their clients.
// PackageID is set when the plan came from a package (NULL = assigned manually).
type AssignedPlanInfo struct {
	UserID    uuid.UUID  `db:"user_id" json:"-"`
	PlanID    uuid.UUID  `db:"plan_id" json:"plan_id"`
	PlanName  string     `db:"plan_name" json:"plan_name"`
	PackageID *uuid.UUID `db:"package_id" json:"package_id,omitempty"`
	CreatedAt time.Time  `db:"created_at" json:"assigned_at"`
}

// ClientPackageInfo is a package a client is subscribed to under the coach.
type ClientPackageInfo struct {
	ClientID    uuid.UUID `db:"client_id" json:"-"`
	PackageID   uuid.UUID `db:"package_id" json:"package_id"`
	PackageName string    `db:"package_name" json:"package_name"`
	CreatedAt   time.Time `db:"created_at" json:"subscribed_at"`
}

// CoachClient is a user enrolled in one of the coach's packages, enriched with
// the packages they're subscribed to and the plans the coach assigned to them.
type CoachClient struct {
	User
	Packages      []ClientPackageInfo `json:"packages"`
	AssignedPlans []AssignedPlanInfo  `json:"assigned_plans"`
}

// ListCoachClients returns the coach's clients — users enrolled in one of the
// coach's packages (a plain connection is NOT a client). Each is enriched with
// their package subscriptions and assigned plans, all gathered in batch (no N+1).
func ListCoachClients(ctx context.Context, coachID uuid.UUID, p database.Paginate) ([]CoachClient, int, error) {
	var (
		clients   = []CoachClient{}
		fetchList []database.FetchList
		ids       []interface{}
		total     int
	)

	if err := database.QuerySelect("subscriptions/list_clients", &fetchList, coachID, p.Limit, p.Offset); err != nil {
		return nil, 0, err
	}
	if len(fetchList) < 1 {
		return clients, 0, nil
	}
	total = fetchList[0].TotalCount
	for _, f := range fetchList {
		ids = append(ids, f.ID)
	}

	var users []User
	if err := database.Fetch(&users, ids...); err != nil {
		return nil, 0, err
	}
	userMap := make(map[uuid.UUID]*User, len(users))
	for i := range users {
		userMap[users[i].ID] = &users[i]
	}

	var assigned []AssignedPlanInfo
	if err := database.QuerySelect("coaches/client_plans", &assigned, coachID); err != nil {
		return nil, 0, err
	}
	plansByUser := make(map[uuid.UUID][]AssignedPlanInfo, len(assigned))
	for _, a := range assigned {
		plansByUser[a.UserID] = append(plansByUser[a.UserID], a)
	}

	var subs []ClientPackageInfo
	if err := database.QuerySelect("subscriptions/client_packages", &subs, coachID); err != nil {
		return nil, 0, err
	}
	packagesByUser := make(map[uuid.UUID][]ClientPackageInfo, len(subs))
	for _, s := range subs {
		packagesByUser[s.ClientID] = append(packagesByUser[s.ClientID], s)
	}

	// Preserve the order from list_clients (most recent first).
	for _, f := range fetchList {
		u, ok := userMap[f.ID]
		if !ok {
			continue
		}
		client := CoachClient{User: *u, Packages: []ClientPackageInfo{}, AssignedPlans: []AssignedPlanInfo{}}
		if pkgs, ok := packagesByUser[u.ID]; ok {
			client.Packages = pkgs
		}
		if plans, ok := plansByUser[u.ID]; ok {
			client.AssignedPlans = plans
		}
		clients = append(clients, client)
	}
	return clients, total, nil
}

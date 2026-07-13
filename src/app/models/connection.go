package models

import (
	"context"
	"errors"
	"time"

	"coachwise/src/database"

	"github.com/google/uuid"
)

type ConnectionRequest struct {
	ID          uuid.UUID `db:"id" json:"id"`
	RequesterID uuid.UUID `db:"requester_id" json:"requester_id"`
	AddresseeID uuid.UUID `db:"addressee_id" json:"addressee_id"`
	Status      string    `db:"status" json:"status"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// ConnectionRequestWithUser is an incoming request plus the hydrated requester.
type ConnectionRequestWithUser struct {
	ID        uuid.UUID `json:"id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	User      *User     `json:"user"`
}

// SendConnectionRequest creates or re-opens a PENDING request from requester to addressee.
func SendConnectionRequest(ctx context.Context, requesterID, addresseeID uuid.UUID) (*ConnectionRequest, error) {
	cr := new(ConnectionRequest)
	rows, err := database.Query(ctx, "connections/request", requesterID, addresseeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.StructScan(cr); err != nil {
			return nil, err
		}
	}
	return cr, nil
}

// CancelConnectionRequest removes the request the requester sent to addressee.
func CancelConnectionRequest(ctx context.Context, requesterID, addresseeID uuid.UUID) error {
	rows, err := database.Query(ctx, "connections/cancel", requesterID, addresseeID)
	if err != nil {
		return err
	}
	rows.Close()
	return nil
}

// AcceptConnectionRequest accepts request id (addressee only) and establishes the connection.
func AcceptConnectionRequest(ctx context.Context, requestID, addresseeID uuid.UUID) (*ConnectionRequest, error) {
	cr := new(ConnectionRequest)
	rows, err := database.Query(ctx, "connections/accept", requestID, addresseeID)
	if err != nil {
		return nil, err
	}
	found := false
	for rows.Next() {
		if err := rows.StructScan(cr); err != nil {
			rows.Close()
			return nil, err
		}
		found = true
	}
	rows.Close()
	if !found {
		return nil, errors.New("connection request not found")
	}

	crows, err := database.Query(ctx, "connections/create", cr.RequesterID, cr.AddresseeID, cr.ID)
	if err != nil {
		return nil, err
	}
	crows.Close()
	return cr, nil
}

// RejectConnectionRequest marks request id REJECTED (addressee only); the row is kept.
func RejectConnectionRequest(ctx context.Context, requestID, addresseeID uuid.UUID) error {
	rows, err := database.Query(ctx, "connections/reject", requestID, addresseeID)
	if err != nil {
		return err
	}
	rows.Close()
	return nil
}

// IsConnected reports whether two users have an established connection.
func IsConnected(ctx context.Context, a, b uuid.UUID) (bool, error) {
	rows, err := database.Query(ctx, "connections/check", a, b)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	if rows.Next() {
		var connected bool
		if err := rows.Scan(&connected); err != nil {
			return false, err
		}
		return connected, nil
	}
	return false, nil
}

type connectionStatusRow struct {
	ID               uuid.UUID `db:"id"`
	IsConnected      bool      `db:"is_connected"`
	ConnectionStatus string    `db:"connection_status"`
}

// SetConnectionStatuses fills ConnectionStatus + IsConnected on each user relative
// to viewerID. The status string and boolean are computed in SQL; Go only maps the
// result rows onto users by ID.
func SetConnectionStatuses(ctx context.Context, viewerID uuid.UUID, users []User) error {
	var rows []connectionStatusRow
	if err := database.QuerySelect("connections/statuses", &rows, viewerID); err != nil {
		return err
	}

	byID := make(map[uuid.UUID]connectionStatusRow, len(rows))
	for _, r := range rows {
		byID[r.ID] = r
	}

	for i := range users {
		if r, ok := byID[users[i].ID]; ok {
			users[i].IsConnected = r.IsConnected
			users[i].ConnectionStatus = r.ConnectionStatus
		} else {
			users[i].ConnectionStatus = "none"
		}
	}
	return nil
}

// ListConnectionsPaginated returns the user's established connections.
func ListConnectionsPaginated(ctx context.Context, viewerID uuid.UUID, p database.Paginate) ([]User, int, error) {
	var (
		items     = []User{}
		fetchList []database.FetchList
		ids       []interface{}
		total     int
	)

	if err := database.QuerySelect("connections/list", &fetchList, viewerID, p.Limit, p.Offset); err != nil {
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

type incomingRequestRow struct {
	ID          uuid.UUID `db:"id"`
	RequesterID uuid.UUID `db:"requester_id"`
	Status      string    `db:"status"`
	CreatedAt   time.Time `db:"created_at"`
	TotalCount  int       `db:"total_count"`
}

// ListIncomingRequestsPaginated returns requests addressed to the user filtered by
// status (PENDING or REJECTED), each with the hydrated requester.
func ListIncomingRequestsPaginated(ctx context.Context, addresseeID uuid.UUID, status string, p database.Paginate) ([]ConnectionRequestWithUser, int, error) {
	var rows []incomingRequestRow
	if err := database.QuerySelect("connections/requests_incoming", &rows, addresseeID, status, p.Limit, p.Offset); err != nil {
		return nil, 0, err
	}
	if len(rows) < 1 {
		return []ConnectionRequestWithUser{}, 0, nil
	}
	total := rows[0].TotalCount

	var ids []interface{}
	for _, r := range rows {
		ids = append(ids, r.RequesterID)
	}
	var users []User
	if err := database.Fetch(&users, ids...); err != nil {
		return nil, 0, err
	}
	userMap := make(map[uuid.UUID]*User, len(users))
	for i := range users {
		userMap[users[i].ID] = &users[i]
	}

	out := make([]ConnectionRequestWithUser, 0, len(rows))
	for _, r := range rows {
		out = append(out, ConnectionRequestWithUser{
			ID:        r.ID,
			Status:    r.Status,
			CreatedAt: r.CreatedAt,
			User:      userMap[r.RequesterID],
		})
	}
	return out, total, nil
}

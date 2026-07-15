package models

import (
	"context"
	"errors"
	"time"

	"coachwise/src/database"
)

// ErrNothingToRefund means the subscription was never paid for (a coach handed
// the package over by hand) or the order has already been refunded in full.
var ErrNothingToRefund = errors.New("nothing to refund")

// Refund is what a cancellation gave back, and why.
type Refund struct {
	Amount        int64  `json:"amount"`        // to the client
	FromCoach     int64  `json:"from_coach"`    // taken back from the coach
	FromPlatform  int64  `json:"from_platform"` // taken back from us (fee + Pro)
	Currency      string `json:"currency"`
	Full          bool   `json:"full"`           // cancelled inside the cooling-off period
	DaysRemaining int    `json:"days_remaining"` // of the term, at cancellation
}

// RefundCancellation gives the client back the part of the term they won't get,
// because the coach ended it.
//
// The coach chose to drop a paying client, so the coach carries the cost. Inside
// the cooling-off window (escrow_hold_days) the money is still held and hasn't
// reached anyone, so the whole purchase is returned. After it, only the unused
// days are, taken proportionally from the coach's net and from our fee — nobody
// delivered the rest, so nobody keeps their cut of it.
//
// Pro is refunded in money but never revoked: it's cheap, and taking back
// features someone has been using would be petty.
//
// The clawback can leave a coach's balance negative if they've already withdrawn.
// That's allowed and honest — payouts are blocked until it's settled — rather
// than pretending the money is still there.
func RefundCancellation(ctx context.Context, sub *PackageSubscription) (*Refund, error) {
	if sub.OrderID == nil {
		return nil, ErrNothingToRefund // handed over by a coach; no money changed hands
	}

	order := new(Order)
	if err := database.Get(order, "orders/fetch", *sub.OrderID); err != nil {
		return nil, err
	}
	refundable := order.Total - order.RefundedAmount
	if refundable <= 0 {
		return nil, ErrNothingToRefund
	}

	settings, err := GetPlatformSettings()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	coolingOffEnds := order.CreatedAt.UTC().AddDate(0, 0, settings.EscrowHoldDays)
	full := now.Before(coolingOffEnds)

	amount := refundable
	daysRemaining := 0
	if !full {
		// Past the cooling-off period: give back the days they won't get. The term
		// runs from the purchase to ends_at, so the share is what's left of it.
		if sub.EndsAt == nil {
			return nil, ErrNothingToRefund
		}
		termEnd := sub.EndsAt.UTC()
		termStart := order.CreatedAt.UTC()
		total := termEnd.Sub(termStart)
		left := termEnd.Sub(now)
		if left <= 0 || total <= 0 {
			return nil, ErrNothingToRefund // the term is spent; they got what they paid for
		}
		if left > total {
			left = total
		}
		daysRemaining = int(left.Hours() / 24)

		// Scale by MINUTES, not by the raw Durations. A Duration is nanoseconds, so
		// `total * int64(left)` is (price × 7.9e15) — which overflows int64 and
		// silently produces a garbage (often negative) refund. Minutes keep the
		// arithmetic exact with room to spare.
		leftMin := int64(left.Minutes())
		totalMin := int64(total.Minutes())
		if totalMin <= 0 {
			return nil, ErrNothingToRefund
		}
		amount = order.Total * leftMin / totalMin
		if amount > refundable {
			amount = refundable
		}
		if amount <= 0 {
			return nil, ErrNothingToRefund
		}
	} else if sub.EndsAt != nil {
		daysRemaining = int(sub.EndsAt.UTC().Sub(now).Hours() / 24)
	}

	// Split it back to where it came from. The coach's share is proportional to
	// their cut of the original order; whatever is left is ours (fee + Pro).
	fromCoach := int64(0)
	if order.Total > 0 {
		fromCoach = order.CoachNet * amount / order.Total
	}
	fromPlatform := amount - fromCoach

	tx, err := database.GetDB().Beginx()
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	buyerWallet, err := txGetOrCreateUserWallet(ctx, tx, order.BuyerID, order.Currency)
	if err != nil {
		return nil, err
	}
	// The client gets it back now — no escrow on money coming home.
	if err := addEntry(ctx, tx, buyerWallet.ID, order.Currency, amount, "REFUND", now, order.ID, "Refund: subscription cancelled"); err != nil {
		return nil, err
	}

	if fromCoach > 0 && order.CoachID != nil {
		coachWallet, err := txGetOrCreateUserWallet(ctx, tx, *order.CoachID, order.Currency)
		if err != nil {
			return nil, err
		}
		if err := addEntry(ctx, tx, coachWallet.ID, order.Currency, -fromCoach, "REFUND", now, order.ID, "Refund: you cancelled a client"); err != nil {
			return nil, err
		}
	}
	if fromPlatform > 0 {
		platform, err := txGetOrCreatePlatformWallet(ctx, tx, order.Currency)
		if err != nil {
			return nil, err
		}
		if err := addEntry(ctx, tx, platform.ID, order.Currency, -fromPlatform, "REFUND", now, order.ID, "Refund: subscription cancelled"); err != nil {
			return nil, err
		}
	}

	if err := txExec(ctx, tx, "orders/record_refund", order.ID, amount); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true

	return &Refund{
		Amount:        amount,
		FromCoach:     fromCoach,
		FromPlatform:  fromPlatform,
		Currency:      order.Currency,
		Full:          full,
		DaysRemaining: daysRemaining,
	}, nil
}

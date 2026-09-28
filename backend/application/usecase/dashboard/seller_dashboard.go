// Package dashboard implements the aggregated read models the farmer
// portal and admin console render — numbers computed from reservation,
// cycle and user data that already exists, never a separate stored
// "stats" collection that could drift from it.
package dashboard

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// SellerDashboard backs the farmer portal's "Tableau de bord" — every
// figure here is computed from the same reservation/cycle data the
// farmer's other pages read, never a separately maintained counter that
// could drift from it.
type SellerDashboard struct {
	Farms        ports.FarmRepository
	Cycles       ports.CycleStore
	Listings     ports.ListingStore
	Reservations ports.ReservationRepository
	Users        ports.UserRepository
	Clock        ports.Clock
}

type SellerDashboardInput struct {
	FarmID  bson.ObjectID
	OwnerID bson.ObjectID
}

type CycleSummary struct {
	Cycle             *domain.Cycle
	FillRate          float64 // 0-1: (capacity-available)/capacity
	NextMilestone     string  // "Éclosion", "Début", "Sortie" — the design's own vocabulary, kept generic here
	NextMilestoneDate time.Time
}

type SellerDashboardOutput struct {
	TrustScore float64

	ReservationsThisPeriod int
	EscrowHeldTotal        float64
	ReleasedThisPeriod     float64

	// WeeklyReservationCounts covers the last 8 weeks, oldest first —
	// the design's own bar chart window.
	WeeklyReservationCounts [8]int

	ActiveCycles       []CycleSummary
	LatestReservations []*domain.Reservation
}

const latestReservationsLimit = 6

// Execute returns domain.ErrForbidden if the caller doesn't own the
// farm.
func (uc *SellerDashboard) Execute(ctx context.Context, in SellerDashboardInput) (*SellerDashboardOutput, error) {
	f, err := uc.Farms.FindByID(ctx, in.FarmID)
	if err != nil {
		return nil, err
	}
	if f.OwnerID != in.OwnerID {
		return nil, fmt.Errorf("dashboard: not the farm owner: %w", domain.ErrForbidden)
	}

	cycles, err := uc.Cycles.ListByFarm(ctx, in.FarmID)
	if err != nil {
		return nil, err
	}

	listingIDs := make([]bson.ObjectID, 0, len(cycles))
	cycleByListing := map[bson.ObjectID]*domain.Cycle{}
	for _, c := range cycles {
		l, err := uc.Listings.FindByCycleID(ctx, c.ID)
		if errors.Is(err, domain.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		listingIDs = append(listingIDs, l.ID)
		cycleByListing[l.ID] = c
	}

	reservations, err := uc.Reservations.ListByListingIDs(ctx, listingIDs)
	if err != nil {
		return nil, err
	}

	now := uc.Clock.Now()
	out := &SellerDashboardOutput{}

	if u, err := uc.Users.FindByID(ctx, in.OwnerID); err == nil {
		out.TrustScore = u.TrustScore
	}

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	for _, r := range reservations {
		if !r.CreatedAt.Before(monthStart) {
			out.ReservationsThisPeriod++
		}
		if r.Payment != nil {
			switch r.Payment.Status {
			case domain.PaymentHeld:
				out.EscrowHeldTotal += r.TotalAmount
			case domain.PaymentReleased:
				if r.Payment.ReleasedAt != nil && !r.Payment.ReleasedAt.Before(monthStart) {
					out.ReleasedThisPeriod += r.TotalAmount
				}
			}
		}

		if weekIndex, ok := weekBucket(now, r.CreatedAt); ok {
			out.WeeklyReservationCounts[weekIndex]++
		}
	}

	out.LatestReservations = latestN(reservations, latestReservationsLimit)

	for _, c := range cycles {
		if c.Status != domain.CycleStatusPlanned && c.Status != domain.CycleStatusInProgress {
			continue
		}
		summary := CycleSummary{Cycle: c}
		if c.Capacity > 0 {
			summary.FillRate = float64(c.Capacity-c.AvailableQuantity) / float64(c.Capacity)
		}
		if c.Status == domain.CycleStatusPlanned {
			summary.NextMilestone, summary.NextMilestoneDate = "Début", c.StartsAt
		} else if c.EndsAt != nil {
			summary.NextMilestone, summary.NextMilestoneDate = "Sortie", *c.EndsAt
		}
		out.ActiveCycles = append(out.ActiveCycles, summary)
	}

	return out, nil
}

// weekBucket places t into one of the 8 weekly buckets ending at now,
// oldest first (bucket 7 is the current week) — returns ok=false for
// anything older than 8 weeks ago.
func weekBucket(now, t time.Time) (int, bool) {
	const week = 7 * 24 * time.Hour
	age := now.Sub(t)
	if age < 0 || age >= 8*week {
		return 0, false
	}
	return 7 - int(age/week), true
}

func latestN(reservations []*domain.Reservation, n int) []*domain.Reservation {
	sorted := make([]*domain.Reservation, len(reservations))
	copy(sorted, reservations)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j].CreatedAt.After(sorted[j-1].CreatedAt); j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	if len(sorted) > n {
		sorted = sorted[:n]
	}
	return sorted
}

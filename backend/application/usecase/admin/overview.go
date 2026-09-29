package admin

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// Overview backs the admin console's landing page — every figure is a
// real aggregation over the same collections every other module writes
// to, never a separately maintained counter.
type Overview struct {
	Cycles       ports.CycleStore
	Listings     ports.ListingStore
	Reservations ports.ReservationRepository
	Reports      ports.ReportRepository
	Clock        ports.Clock
}

type OverviewOutput struct {
	VolumeReserved         float64
	EscrowHeld             float64
	ActiveCyclesPlanned    int
	ActiveCyclesInProgress int
	OpenReports            int
	// ReservationsPerDay covers the last 30 days, oldest first.
	ReservationsPerDay [30]int
}

func (uc *Overview) Execute(ctx context.Context) (*OverviewOutput, error) {
	out := &OverviewOutput{}

	var allCycles []*domain.Cycle
	for _, status := range []domain.CycleStatus{
		domain.CycleStatusPlanned, domain.CycleStatusInProgress, domain.CycleStatusDone, domain.CycleStatusCancelled,
	} {
		cycles, err := uc.Cycles.ListByStatus(ctx, status)
		if err != nil {
			return nil, err
		}
		allCycles = append(allCycles, cycles...)
		switch status {
		case domain.CycleStatusPlanned:
			out.ActiveCyclesPlanned = len(cycles)
		case domain.CycleStatusInProgress:
			out.ActiveCyclesInProgress = len(cycles)
		}
	}

	openReports, err := uc.Reports.List(ctx, domain.ReportOpen)
	if err != nil {
		return nil, err
	}
	underReview, err := uc.Reports.List(ctx, domain.ReportUnderReview)
	if err != nil {
		return nil, err
	}
	out.OpenReports = len(openReports) + len(underReview)

	listingIDs := make([]bson.ObjectID, 0, len(allCycles))
	for _, c := range allCycles {
		l, err := uc.Listings.FindByCycleID(ctx, c.ID)
		if errors.Is(err, domain.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		listingIDs = append(listingIDs, l.ID)
	}
	reservations, err := uc.Reservations.ListByListingIDs(ctx, listingIDs)
	if err != nil {
		return nil, err
	}

	now := uc.Clock.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	for _, r := range reservations {
		out.VolumeReserved += r.TotalAmount
		if r.Payment != nil && r.Payment.Status == domain.PaymentHeld {
			out.EscrowHeld += r.TotalAmount
		}
		age := dayStart.Sub(time.Date(r.CreatedAt.Year(), r.CreatedAt.Month(), r.CreatedAt.Day(), 0, 0, 0, 0, r.CreatedAt.Location()))
		day := int(age.Hours() / 24)
		if day >= 0 && day < 30 {
			out.ReservationsPerDay[29-day]++
		}
	}

	return out, nil
}

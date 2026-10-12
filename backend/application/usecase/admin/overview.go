package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// Period is the window the overview's activity figures cover.
type Period string

const (
	Period7Days   Period = "7d"
	Period30Days  Period = "30d"
	Period12Month Period = "12m"
)

// Overview backs the back office's landing page — every figure is a
// real aggregation over the same collections every other module writes
// to, never a separately maintained counter.
type Overview struct {
	Cycles       ports.CycleStore
	Listings     ports.ListingStore
	Reservations ports.ReservationRepository
	Reports      ports.ReportRepository
	Farms        ports.FarmRepository
	Clock        ports.Clock
	// AutoApproveFarms and MaxFarms are the deployment's switches
	// (FARM_AUTO_APPROVE, MAX_FARMS_PER_ACCOUNT), shown so staff know
	// how the platform is currently configured.
	AutoApproveFarms bool
	MaxFarms         int
}

// OverviewBucket is one bar of the reservations chart: a day, or a
// month for the 12-month view.
type OverviewBucket struct {
	Start time.Time
	// Paid reservations hold places; Waiting ones are on a waiting list.
	Paid    int
	Waiting int
}

type OverviewOutput struct {
	Period Period
	// VolumeReserved is what was reserved during the period (cancelled
	// reservations left out); VolumePrevious the same over the period
	// just before, for the trend.
	VolumeReserved float64
	VolumePrevious float64
	// EscrowHeld and HeldReservations are what the platform holds now.
	EscrowHeld             float64
	HeldReservations       int
	ActiveCyclesPlanned    int
	ActiveCyclesInProgress int
	// OpenReports leaves disputes out; OpenDisputes counts them.
	OpenReports  int
	OpenDisputes int
	// PendingFarms wait for a decision; OldestPendingSince is when the
	// oldest of them was submitted.
	PendingFarms       int
	OldestPendingSince *time.Time
	Series             []OverviewBucket

	AutoApproveFarms bool
	MaxFarms         int
}

func (uc *Overview) Execute(ctx context.Context, period Period) (*OverviewOutput, error) {
	if period == "" {
		period = Period30Days
	}
	now := uc.Clock.Now()
	buckets, previousStart, err := periodBuckets(period, now)
	if err != nil {
		return nil, err
	}
	out := &OverviewOutput{Period: period, Series: buckets, AutoApproveFarms: uc.AutoApproveFarms, MaxFarms: uc.MaxFarms}
	if out.MaxFarms <= 0 {
		out.MaxFarms = domain.DefaultMaxFarmsPerAccount
	}

	for _, status := range []domain.CycleStatus{domain.CycleStatusPlanned, domain.CycleStatusInProgress} {
		cycles, err := uc.Cycles.ListByStatus(ctx, status)
		if err != nil {
			return nil, err
		}
		if status == domain.CycleStatusPlanned {
			out.ActiveCyclesPlanned = len(cycles)
		} else {
			out.ActiveCyclesInProgress = len(cycles)
		}
	}

	counts, err := countOpenReports(ctx, uc.Reports)
	if err != nil {
		return nil, err
	}
	out.OpenReports, out.OpenDisputes = counts.reports, counts.disputes

	if uc.Farms != nil {
		pending, err := uc.Farms.ListByStatus(ctx, domain.FarmPending)
		if err != nil {
			return nil, err
		}
		out.PendingFarms = len(pending)
		// ListByStatus is oldest submission first.
		if len(pending) > 0 {
			out.OldestPendingSince = pending[0].Verification.SubmittedAt
		}
	}

	listingIDs, err := allListingIDs(ctx, uc.Cycles, uc.Listings)
	if err != nil {
		return nil, err
	}
	reservations, err := uc.Reservations.ListByListingIDs(ctx, listingIDs)
	if err != nil {
		return nil, err
	}

	periodStart := buckets[0].Start
	for _, r := range reservations {
		paid := r.Payment != nil && (r.Payment.Status == domain.PaymentHeld || r.Payment.Status == domain.PaymentReleased)
		if r.Payment != nil && r.Payment.Status == domain.PaymentHeld {
			out.EscrowHeld += r.TotalAmount
			out.HeldReservations++
		}
		if r.Status == domain.ReservationCancelled {
			continue
		}
		switch {
		case !r.CreatedAt.Before(periodStart):
			out.VolumeReserved += r.TotalAmount
		case !r.CreatedAt.Before(previousStart):
			out.VolumePrevious += r.TotalAmount
		}
		for i := len(buckets) - 1; i >= 0; i-- {
			if !r.CreatedAt.Before(buckets[i].Start) {
				if paid {
					buckets[i].Paid++
				} else {
					buckets[i].Waiting++
				}
				break
			}
		}
	}
	return out, nil
}

// periodBuckets lays out the chart's bars, oldest first, the last one
// holding now, and returns where the preceding period of the same
// length starts.
func periodBuckets(period Period, now time.Time) ([]OverviewBucket, time.Time, error) {
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch period {
	case Period7Days, Period30Days:
		n := 7
		if period == Period30Days {
			n = 30
		}
		buckets := make([]OverviewBucket, n)
		for i := range buckets {
			buckets[i].Start = day.AddDate(0, 0, i-(n-1))
		}
		return buckets, buckets[0].Start.AddDate(0, 0, -n), nil
	case Period12Month:
		month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		buckets := make([]OverviewBucket, 12)
		for i := range buckets {
			buckets[i].Start = month.AddDate(0, i-11, 0)
		}
		return buckets, buckets[0].Start.AddDate(-1, 0, 0), nil
	}
	return nil, time.Time{}, fmt.Errorf("admin: unknown period %q: %w", period, domain.ErrInvalid)
}

type openReportCounts struct{ reports, disputes int }

// countOpenReports splits what still needs staff into reports (about a
// listing, a profile, a forum post) and disputes (about a reservation).
func countOpenReports(ctx context.Context, reports ports.ReportRepository) (openReportCounts, error) {
	var out openReportCounts
	for _, status := range []domain.ReportStatus{domain.ReportOpen, domain.ReportUnderReview} {
		list, err := reports.List(ctx, status)
		if err != nil {
			return out, err
		}
		for _, rp := range list {
			if rp.TargetType == domain.ReportTargetReservation {
				out.disputes++
			} else {
				out.reports++
			}
		}
	}
	return out, nil
}

// NavCounts is what the back office's sidebar badges show: work
// waiting for staff.
type NavCounts struct {
	PendingFarms int
	OpenReports  int
	OpenDisputes int
}

// CountNav backs those badges.
type CountNav struct {
	Farms   ports.FarmRepository
	Reports ports.ReportRepository
}

func (uc *CountNav) Execute(ctx context.Context) (*NavCounts, error) {
	pending, err := uc.Farms.ListByStatus(ctx, domain.FarmPending)
	if err != nil {
		return nil, err
	}
	counts, err := countOpenReports(ctx, uc.Reports)
	if err != nil {
		return nil, err
	}
	return &NavCounts{PendingFarms: len(pending), OpenReports: counts.reports, OpenDisputes: counts.disputes}, nil
}

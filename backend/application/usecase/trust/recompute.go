// Package trust implements the one trust-score computation the
// architecture document describes as "signals combined" rather than a
// single boolean: profile verification, successful-transaction history,
// verified reviews, and community reports. Other modules trigger a
// recompute through ports.TrustRecomputer — see that port's doc comment
// for why this package is never imported directly by another usecase
// package.
package trust

import (
	"context"
	"slices"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/farmstats"
	"github.com/pauluno777/betailo/backend/domain"
)

// Score weights. Chosen to keep the formula legible and explainable in
// one paragraph rather than tuned against real data (there is none yet)
// — see the package doc for the signals they encode.
const (
	baseScore              = 50.0
	verifiedIdentityBonus  = 15.0
	maxSalesBonus          = 30.0
	salesBonusPerSale      = 1.5
	maxReviewBonus         = 20.0
	reviewConfidenceFloor  = 5 // fewer than this many reviews and the bonus is scaled down, not full weight
	reportPenaltyPerReport = 10.0
	reportLookback         = 30 * 24 * time.Hour
)

var _ ports.TrustRecomputer = (*Recompute)(nil)

// Recompute is the usecase: gather every signal for a user and persist
// the resulting score. Cheap enough to call synchronously right after
// the event that changed a signal (a review posted, escrow released, a
// report filed, an admin verification) — see its callers in
// application/usecase/reservation, review, and admin.
type Recompute struct {
	Users        ports.UserRepository
	Farms        ports.FarmRepository
	Cycles       ports.CycleStore
	Listings     ports.ListingStore
	Reservations ports.ReservationRepository
	Reviews      ports.ReviewRepository
	Reports      ports.ReportRepository
	Clock        ports.Clock
}

// Execute implements ports.TrustRecomputer.
func (uc *Recompute) Execute(ctx context.Context, userID bson.ObjectID) error {
	u, err := uc.Users.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	farms, err := uc.Farms.ListByOwner(ctx, userID)
	if err != nil {
		return err
	}
	score := baseScore
	// The identity bonus now comes with a farm staff reviewed: that
	// review is where the owner's identity document is checked.
	if slices.ContainsFunc(farms, func(f *domain.Farm) bool { return f.Status.CanSell() && !f.Verification.AutoApproved }) {
		score += verifiedIdentityBonus
	}

	sales, err := uc.releasedSalesCount(ctx, userID)
	if err != nil {
		return err
	}
	score += min(float64(sales)*salesBonusPerSale, maxSalesBonus)

	avg, count, err := uc.reviewSignal(ctx, userID)
	if err != nil {
		return err
	}
	if count > 0 {
		// (avg-3) spans [-2, 2] for a 1-5 rating; scaled to
		// [-maxReviewBonus, maxReviewBonus] and damped for a thin
		// sample so five 5-star reviews don't outweigh fifty averaging
		// 4.5.
		confidence := min(float64(count)/reviewConfidenceFloor, 1.0)
		score += (avg - 3) * (maxReviewBonus / 2) * confidence
	}

	reportCount, err := uc.Reports.CountRecentByTarget(ctx, userID, uc.Clock.Now().Add(-reportLookback))
	if err != nil {
		return err
	}
	score -= float64(reportCount) * reportPenaltyPerReport

	u.TrustScore = clamp(score, 0, 100)
	return uc.Users.Update(ctx, u)
}

// releasedSalesCount counts released-escrow sales across every farm
// this user owns. See farmstats.ReleasedSales for what counts as a sale.
func (uc *Recompute) releasedSalesCount(ctx context.Context, userID bson.ObjectID) (int, error) {
	farms, err := uc.Farms.ListByOwner(ctx, userID)
	if err != nil {
		return 0, err
	}
	ids := make([]bson.ObjectID, 0, len(farms))
	for _, f := range farms {
		ids = append(ids, f.ID)
	}
	return farmstats.ReleasedSales(ctx, farmstats.Sources{Cycles: uc.Cycles, Listings: uc.Listings, Reservations: uc.Reservations}, ids...)
}

// reviewSignal aggregates ReviewRepository.Summary across every farm
// this user owns — a user, not a farm, carries the trust score, so a
// multi-farm owner's reviews all feed the same number.
func (uc *Recompute) reviewSignal(ctx context.Context, userID bson.ObjectID) (avg float64, count int, err error) {
	farms, err := uc.Farms.ListByOwner(ctx, userID)
	if err != nil {
		return 0, 0, err
	}

	var weightedSum float64
	for _, f := range farms {
		s, err := uc.Reviews.Summary(ctx, f.ID)
		if err != nil {
			return 0, 0, err
		}
		weightedSum += s.Average * float64(s.Count)
		count += s.Count
	}
	if count == 0 {
		return 0, 0, nil
	}
	return weightedSum / float64(count), count, nil
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

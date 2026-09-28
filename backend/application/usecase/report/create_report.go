// Package report implements the anti-scam reporting flow: any
// authenticated user can flag a listing, profile or forum post, and
// repeated reports against the same target trip an automatic
// under-review state — exactly the sequence diagram the "Concepts &
// Architecture" document describes for "Signalement anti-scam et
// reaction automatique".
package report

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// RepeatedReportThreshold and RepeatedReportWindow are the architecture
// document's own numbers ("3 signalements en 7 jours").
const (
	RepeatedReportThreshold = 3
	RepeatedReportWindow    = 7 * 24 * time.Hour
)

// CreateReport is the usecase. Disputing a reservation's escrow has its
// own flow (application/usecase/reservation.OpenDispute) — this one
// only accepts the three target types a marketplace visitor can
// actually report from the UI.
type CreateReport struct {
	Reports  ports.ReportRepository
	Listings ports.ListingStore
	Clock    ports.Clock
}

type CreateReportInput struct {
	ReporterID bson.ObjectID
	TargetID   bson.ObjectID
	TargetType domain.ReportTargetType
	Reason     string
}

// Execute returns domain.ErrInvalid for an empty reason or an
// unsupported target type.
func (uc *CreateReport) Execute(ctx context.Context, in CreateReportInput) (*domain.Report, error) {
	switch in.TargetType {
	case domain.ReportTargetListing, domain.ReportTargetProfile, domain.ReportTargetForumPost:
	default:
		return nil, fmt.Errorf("report: unsupported target type %q: %w", in.TargetType, domain.ErrInvalid)
	}
	if in.Reason == "" {
		return nil, fmt.Errorf("report: a reason is required: %w", domain.ErrInvalid)
	}

	now := uc.Clock.Now()
	rp := &domain.Report{
		ID: bson.NewObjectID(), ReporterID: in.ReporterID, TargetID: in.TargetID,
		TargetType: in.TargetType, Reason: in.Reason, Status: domain.ReportOpen, CreatedAt: now,
	}
	if err := uc.Reports.Insert(ctx, rp); err != nil {
		return nil, err
	}

	count, err := uc.Reports.CountRecentByTarget(ctx, in.TargetID, now.Add(-RepeatedReportWindow))
	if err != nil {
		return rp, err
	}
	if count < RepeatedReportThreshold {
		return rp, nil
	}

	rp.Status = domain.ReportUnderReview
	if err := uc.Reports.Update(ctx, rp); err != nil {
		return rp, err
	}

	// Only a listing gets an automatic visibility change — a reported
	// profile or forum post waits for an admin's explicit decision (see
	// application/usecase/admin), matching the design's moderation
	// console rather than auto-suspending an account on report volume
	// alone.
	if in.TargetType == domain.ReportTargetListing {
		if l, err := uc.Listings.FindByID(ctx, in.TargetID); err == nil {
			l.Flagged = true
			_ = uc.Listings.Update(ctx, l)
		}
	}

	return rp, nil
}

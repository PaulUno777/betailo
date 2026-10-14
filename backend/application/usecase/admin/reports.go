package admin

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// ListReports backs the admin console's "Signalements" queue.
type ListReports struct {
	Reports ports.ReportRepository
}

func (uc *ListReports) Execute(ctx context.Context, status domain.ReportStatus) ([]*domain.Report, error) {
	return uc.Reports.List(ctx, status)
}

// RiskSignal is one line of the admin detail view's "Signaux combinés"
// panel — a named, individually-inspectable reason a target looks risky,
// not just an opaque number.
type RiskSignal struct {
	Name   string
	Detail string
	Weight int
}

// ReportDetail is a report plus the combined risk score the design's
// moderation console shows — every signal is a real backend
// computation (account age, price vs. the marketplace average,
// verification status, report volume), never a client-side guess.
type ReportDetail struct {
	Report    *domain.Report
	History   []*domain.Report
	RiskScore int
	Signals   []RiskSignal
}

// accountAgeThreshold/priceDeviationThreshold mirror the architecture
// document's own example rule: "compte tres recent + prix anormalement
// bas + demande de paiement hors plateforme = signal fort".
const (
	accountAgeThreshold    = 7 * 24 * time.Hour
	priceDeviationFraction = 0.3 // 30% below the species/type average
	riskWeightAccountAge   = 30
	riskWeightPrice        = 30
	riskWeightReports      = 15 // per report, capped
	riskWeightUnverified   = 10
	maxReportsWeight       = 45
)

// GetReportDetail composes a report's history and risk score.
type GetReportDetail struct {
	Reports  ports.ReportRepository
	Users    ports.UserRepository
	Listings ports.ListingStore
	Cycles   ports.CycleStore
	Farms    ports.FarmRepository
	Search   ports.ListingSearchRepository
	Clock    ports.Clock
}

func (uc *GetReportDetail) Execute(ctx context.Context, reportID bson.ObjectID) (*ReportDetail, error) {
	rp, err := uc.Reports.FindByID(ctx, reportID)
	if err != nil {
		return nil, err
	}
	history, err := uc.Reports.ListByTarget(ctx, rp.TargetID)
	if err != nil {
		return nil, err
	}

	detail := &ReportDetail{Report: rp, History: history}
	now := uc.Clock.Now()

	if rp.TargetType == domain.ReportTargetListing {
		if l, err := uc.Listings.FindByID(ctx, rp.TargetID); err == nil {
			uc.scoreListing(ctx, detail, l, now)
		}
	}

	weight := min(riskWeightReports*len(history), maxReportsWeight)
	if weight > 0 {
		detail.Signals = append(detail.Signals, RiskSignal{
			Name: "Signalements récents", Detail: fmt.Sprintf("%d signalement(s)", len(history)), Weight: weight,
		})
		detail.RiskScore += weight
	}

	if detail.RiskScore > 100 {
		detail.RiskScore = 100
	}
	return detail, nil
}

// scoreListing adds the price-vs-market and account-age signals — the
// two the architecture document names explicitly — by comparing the
// listing's price against the marketplace average for the same
// species/type via a search over the current published catalog, and
// the owning account's age against accountAgeThreshold.
func (uc *GetReportDetail) scoreListing(ctx context.Context, detail *ReportDetail, l *domain.Listing, now time.Time) {
	if l.CycleID == nil {
		return
	}

	if c, err := uc.Cycles.FindByID(ctx, *l.CycleID); err == nil {
		if f, err := uc.Farms.FindByID(ctx, c.FarmID); err == nil {
			if owner, err := uc.Users.FindByID(ctx, f.OwnerID); err == nil {
				age := now.Sub(owner.CreatedAt)
				if age < accountAgeThreshold {
					detail.Signals = append(detail.Signals, RiskSignal{
						Name: "Compte très récent", Detail: fmt.Sprintf("créé il y a %.0f j", age.Hours()/24), Weight: riskWeightAccountAge,
					})
					detail.RiskScore += riskWeightAccountAge
				}
			}
			// A listing can only be live from an approved farm, so this
			// fires for one whose farm was since rejected or suspended,
			// or approved by the test switch without a person looking.
			if !f.Status.CanSell() || f.Verification.AutoApproved {
				detail.Signals = append(detail.Signals, RiskSignal{
					Name: "Ferme non vérifiée", Detail: "aucune validation par l'équipe", Weight: riskWeightUnverified,
				})
				detail.RiskScore += riskWeightUnverified
			}
		}
	}

	result, err := uc.Search.Search(ctx, ports.ListingSearchCriteria{PageSize: 100})
	if err == nil {
		var sum float64
		var count int
		var species, cycleType string
		for _, item := range result.Items {
			if item.Listing.ID == l.ID {
				species, cycleType = item.Species, item.CycleType
			}
		}
		for _, item := range result.Items {
			if item.Species == species && item.CycleType == cycleType {
				sum += item.Listing.UnitPrice
				count++
			}
		}
		if count > 0 {
			avg := sum / float64(count)
			if avg > 0 && l.UnitPrice < avg*(1-priceDeviationFraction) {
				detail.Signals = append(detail.Signals, RiskSignal{
					Name: "Prix anormalement bas", Detail: fmt.Sprintf("%.0f vs moyenne %.0f", l.UnitPrice, avg), Weight: riskWeightPrice,
				})
				detail.RiskScore += riskWeightPrice
			}
		}
	}
}

// ResolveAction is what staff decide about a report. (Identity checks
// are no longer an action here: a farm's own review covers them, see
// application/usecase/farm.ReviewFarm.)
type ResolveAction string

const (
	ResolveDismiss ResolveAction = "dismiss"
	ResolveSuspend ResolveAction = "suspend"
)

// ResolveReport applies an admin's moderation decision — only
// meaningful for a profile-targeted report, since suspending only
// makes sense against a user account.
type ResolveReport struct {
	Reports ports.ReportRepository
	Users   ports.UserRepository
	Trust   ports.TrustRecomputer
	// Sessions, when set, ends a suspended user's open sessions at once.
	Sessions SessionForgetter
}

type ResolveReportInput struct {
	ReportID bson.ObjectID
	Action   ResolveAction
}

// Execute returns domain.ErrInvalid for an unrecognized action, or for
// suspend against anything but a profile report.
func (uc *ResolveReport) Execute(ctx context.Context, in ResolveReportInput) (*domain.Report, error) {
	rp, err := uc.Reports.FindByID(ctx, in.ReportID)
	if err != nil {
		return nil, err
	}

	switch in.Action {
	case ResolveDismiss:
		rp.Status = domain.ReportRejected
	case ResolveSuspend:
		if rp.TargetType != domain.ReportTargetProfile {
			return nil, fmt.Errorf("admin: %s only applies to a profile report: %w", in.Action, domain.ErrInvalid)
		}
		u, err := uc.Users.FindByID(ctx, rp.TargetID)
		if err != nil {
			return nil, err
		}
		u.Suspended = true
		if err := uc.Users.Update(ctx, u); err != nil {
			return nil, err
		}
		if u.Suspended && uc.Sessions != nil {
			uc.Sessions.Forget(u.ID)
		}
		_ = uc.Trust.Execute(ctx, u.ID)
		rp.Status = domain.ReportHandled
	default:
		return nil, fmt.Errorf("admin: unrecognized resolve action %q: %w", in.Action, domain.ErrInvalid)
	}

	if err := uc.Reports.Update(ctx, rp); err != nil {
		return nil, err
	}
	return rp, nil
}

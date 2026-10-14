package admin

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// ReportLevel is how urgent a report looks in the queue.
type ReportLevel string

const (
	ReportLevelHigh   ReportLevel = "high"
	ReportLevelMedium ReportLevel = "medium"
	ReportLevelLow    ReportLevel = "low"
)

// highRiskScore is the combined score from which a report is urgent
// whatever its volume.
const highRiskScore = 60

// ReportRow is a report as the moderation queue shows it: what it
// points at by name, who sent it, and how many reports that target has
// drawn. The level is decided here so every screen ranks alike.
type ReportRow struct {
	Report       *domain.Report
	TargetLabel  string
	ReporterName string
	Count        int
	Level        ReportLevel
}

// reportLevel ranks a report from its volume: the automatic review
// threshold, or three reports, is urgent; two is worth a look.
func reportLevel(rp *domain.Report, count int) ReportLevel {
	switch {
	case rp.Status == domain.ReportUnderReview || count >= 3:
		return ReportLevelHigh
	case count == 2:
		return ReportLevelMedium
	default:
		return ReportLevelLow
	}
}

// reportNames resolves the names a report row carries. Every lookup is
// optional: a target deleted since, or a repository left out, leaves the
// label empty rather than failing the queue.
type reportNames struct {
	Users        ports.UserRepository
	Listings     ports.ListingStore
	Reservations ports.ReservationRepository
}

func (n reportNames) user(ctx context.Context, id bson.ObjectID) string {
	if n.Users == nil {
		return ""
	}
	u, err := n.Users.FindByID(ctx, id)
	if err != nil {
		return ""
	}
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.Email
}

func (n reportNames) target(ctx context.Context, rp *domain.Report) string {
	switch rp.TargetType {
	case domain.ReportTargetListing:
		if n.Listings != nil {
			if l, err := n.Listings.FindByID(ctx, rp.TargetID); err == nil {
				return l.Title
			}
		}
	case domain.ReportTargetProfile:
		return n.user(ctx, rp.TargetID)
	case domain.ReportTargetReservation:
		if n.Reservations != nil {
			if r, err := n.Reservations.FindByID(ctx, rp.TargetID); err == nil {
				return r.Reference
			}
		}
	}
	return ""
}

// ListReports backs the admin console's "Signalements" queue.
type ListReports struct {
	Reports      ports.ReportRepository
	Users        ports.UserRepository
	Listings     ports.ListingStore
	Reservations ports.ReservationRepository
}

func (uc *ListReports) Execute(ctx context.Context, status domain.ReportStatus) ([]ReportRow, error) {
	reports, err := uc.Reports.List(ctx, status)
	if err != nil {
		return nil, err
	}
	names := reportNames{Users: uc.Users, Listings: uc.Listings, Reservations: uc.Reservations}
	counts := map[bson.ObjectID]int{}
	labels := map[bson.ObjectID]string{}
	rows := make([]ReportRow, 0, len(reports))
	for _, rp := range reports {
		if _, seen := counts[rp.TargetID]; !seen {
			all, err := uc.Reports.ListByTarget(ctx, rp.TargetID)
			if err != nil {
				return nil, err
			}
			counts[rp.TargetID] = len(all)
			labels[rp.TargetID] = names.target(ctx, rp)
		}
		count := counts[rp.TargetID]
		rows = append(rows, ReportRow{
			Report: rp, TargetLabel: labels[rp.TargetID], ReporterName: names.user(ctx, rp.ReporterID),
			Count: count, Level: reportLevel(rp, count),
		})
	}
	return rows, nil
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
	History   []ReportRow
	RiskScore int
	Signals   []RiskSignal
	Level     ReportLevel
	// TargetLabel names what was reported; Subject is the account behind
	// it, when there is one to act on.
	TargetLabel string
	Subject     *ReportSubject
	Listing     *ReportListing
}

// ReportSubject is the account a report is about: the profile itself,
// or the owner of the reported listing's farm.
type ReportSubject struct {
	ID        bson.ObjectID
	Name      string
	City      string
	CreatedAt time.Time
	Suspended bool
}

// ReportListing is the reported listing as the detail pane previews it.
type ReportListing struct {
	ID        bson.ObjectID
	Title     string
	UnitPrice float64
	Unit      string
	Status    domain.ListingStatus
	PhotoKey  string
	FarmName  string
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
	// Reservations names a reported reservation by its reference.
	Reservations ports.ReservationRepository
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

	names := reportNames{Users: uc.Users, Listings: uc.Listings, Reservations: uc.Reservations}
	rows := make([]ReportRow, 0, len(history))
	for _, h := range history {
		rows = append(rows, ReportRow{Report: h, ReporterName: names.user(ctx, h.ReporterID), Count: len(history), Level: reportLevel(h, len(history))})
	}
	detail := &ReportDetail{Report: rp, History: rows, TargetLabel: names.target(ctx, rp), Level: reportLevel(rp, len(history))}
	now := uc.Clock.Now()

	if rp.TargetType == domain.ReportTargetProfile {
		if u, err := uc.Users.FindByID(ctx, rp.TargetID); err == nil {
			detail.Subject = subjectOf(u)
		}
	}

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
	if detail.RiskScore >= highRiskScore {
		detail.Level = ReportLevelHigh
	}
	return detail, nil
}

func subjectOf(u *domain.User) *ReportSubject {
	name := u.DisplayName
	if name == "" {
		name = u.Email
	}
	return &ReportSubject{ID: u.ID, Name: name, City: u.City, CreatedAt: u.CreatedAt, Suspended: u.Suspended}
}

// scoreListing adds the price-vs-market and account-age signals — the
// two the architecture document names explicitly — by comparing the
// listing's price against the marketplace average for the same
// species/type via a search over the current published catalog, and
// the owning account's age against accountAgeThreshold.
func (uc *GetReportDetail) scoreListing(ctx context.Context, detail *ReportDetail, l *domain.Listing, now time.Time) {
	preview := &ReportListing{ID: l.ID, Title: l.Title, UnitPrice: l.UnitPrice, Unit: l.Unit, Status: l.Status}
	if len(l.Photos) > 0 {
		preview.PhotoKey = l.Photos[0]
	}
	detail.Listing = preview
	if l.CycleID == nil {
		return
	}

	if c, err := uc.Cycles.FindByID(ctx, *l.CycleID); err == nil {
		if f, err := uc.Farms.FindByID(ctx, c.FarmID); err == nil {
			preview.FarmName = f.Name
			if owner, err := uc.Users.FindByID(ctx, f.OwnerID); err == nil {
				detail.Subject = subjectOf(owner)
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

// ResolveReport applies a moderation decision. Suspending acts on an
// account: the reported profile, or the owner of the reported listing's
// farm. Any other report can only be dismissed.
type ResolveReport struct {
	Reports ports.ReportRepository
	Users   ports.UserRepository
	Trust   ports.TrustRecomputer
	// Listings, Cycles and Farms lead from a reported listing to the
	// account that owns it. Without them only a profile report suspends.
	Listings ports.ListingStore
	Cycles   ports.CycleStore
	Farms    ports.FarmRepository
	// Sessions, when set, ends a suspended user's open sessions at once.
	Sessions SessionForgetter
}

type ResolveReportInput struct {
	Actor    Actor
	ReportID bson.ObjectID
	Action   ResolveAction
}

// accountBehind is the account a suspension would hit.
func (uc *ResolveReport) accountBehind(ctx context.Context, rp *domain.Report) (bson.ObjectID, error) {
	switch rp.TargetType {
	case domain.ReportTargetProfile:
		return rp.TargetID, nil
	case domain.ReportTargetListing:
		if uc.Listings == nil || uc.Cycles == nil || uc.Farms == nil {
			break
		}
		l, err := uc.Listings.FindByID(ctx, rp.TargetID)
		if err != nil {
			return bson.NilObjectID, err
		}
		if l.CycleID == nil {
			break
		}
		c, err := uc.Cycles.FindByID(ctx, *l.CycleID)
		if err != nil {
			return bson.NilObjectID, err
		}
		f, err := uc.Farms.FindByID(ctx, c.FarmID)
		if err != nil {
			return bson.NilObjectID, err
		}
		return f.OwnerID, nil
	}
	return bson.NilObjectID, fmt.Errorf("admin: suspend needs a profile or listing report: %w", domain.ErrInvalid)
}

// Execute returns domain.ErrInvalid for an unrecognized action, or for
// suspend on a report with no account behind it.
func (uc *ResolveReport) Execute(ctx context.Context, in ResolveReportInput) (*domain.Report, error) {
	rp, err := uc.Reports.FindByID(ctx, in.ReportID)
	if err != nil {
		return nil, err
	}

	switch in.Action {
	case ResolveDismiss:
		rp.Status = domain.ReportRejected
	case ResolveSuspend:
		accountID, err := uc.accountBehind(ctx, rp)
		if err != nil {
			return nil, err
		}
		u, err := uc.Users.FindByID(ctx, accountID)
		if err != nil {
			return nil, err
		}
		if err := checkSuspendable(in.Actor, u); err != nil {
			return nil, err
		}
		if _, err := setSuspended(ctx, uc.Users, uc.Sessions, u, true); err != nil {
			return nil, err
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

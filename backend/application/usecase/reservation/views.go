package reservation

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// Action is something the viewer can do to a reservation right now. The
// API decides these; a client only renders the buttons it's given.
type Action string

const (
	ActionPay           Action = "pay"
	ActionCancel        Action = "cancel"
	ActionConfirm       Action = "confirm"
	ActionDispute       Action = "dispute"
	ActionReview        Action = "review"
	ActionContact       Action = "contact"
	ActionMarkDelivered Action = "mark_delivered"
)

// Viewer is whose list a view is built for: the same reservation offers
// different actions to its buyer and to the farmer selling.
type Viewer int

const (
	ViewerBuyer Viewer = iota
	ViewerFarmer
)

// View is a reservation with everything its card shows, joined and
// decided server-side: what was reserved and from whom, which tab it
// belongs to, its escrow progress, when escrow auto-releases, and what
// the viewer can do next.
type View struct {
	Reservation *domain.Reservation
	Listing     *domain.Listing
	Cycle       *domain.Cycle
	Farm        *domain.Farm

	Group         domain.ReservationGroup
	Steps         []domain.EscrowStep
	AutoReleaseAt *time.Time
	DisputeOpen   bool
	Reviewed      bool
	Actions       []Action
}

// ViewList is one tab of reservations plus what the tab bar and escrow
// banner need across all tabs.
type ViewList struct {
	Items       []View
	Counts      map[domain.ReservationGroup]int
	EscrowTotal float64 // money currently held in escrow across every reservation
}

// viewSources is the read access building views needs.
type viewSources struct {
	Listings ports.ListingStore
	Cycles   ports.CycleStore
	Farms    ports.FarmRepository
	Reports  ports.ReportRepository
	Reviews  ports.ReviewRepository
}

// viewBuilder joins reservations to their listing, cycle and farm with
// per-request memoization, so a list costs one lookup per distinct
// listing, cycle and farm rather than one per reservation.
type viewBuilder struct {
	src      viewSources
	listings map[bson.ObjectID]*domain.Listing
	cycles   map[bson.ObjectID]*domain.Cycle
	farms    map[bson.ObjectID]*domain.Farm
	reviewed map[bson.ObjectID]map[bson.ObjectID]bool // farm -> reservations reviewed
}

func newViewBuilder(src viewSources) *viewBuilder {
	return &viewBuilder{
		src:      src,
		listings: map[bson.ObjectID]*domain.Listing{},
		cycles:   map[bson.ObjectID]*domain.Cycle{},
		farms:    map[bson.ObjectID]*domain.Farm{},
		reviewed: map[bson.ObjectID]map[bson.ObjectID]bool{},
	}
}

func (b *viewBuilder) list(ctx context.Context, all []*domain.Reservation, group domain.ReservationGroup, viewer Viewer) (*ViewList, error) {
	out := &ViewList{Items: []View{}, Counts: map[domain.ReservationGroup]int{
		domain.GroupActive: 0, domain.GroupDone: 0, domain.GroupCancelled: 0,
	}}
	// Newest first.
	sorted := slices.Clone(all)
	slices.SortFunc(sorted, func(a, b *domain.Reservation) int { return b.CreatedAt.Compare(a.CreatedAt) })

	for _, r := range sorted {
		g := domain.GroupOf(r)
		out.Counts[g]++
		if r.Payment != nil && r.Payment.Status == domain.PaymentHeld {
			out.EscrowTotal += r.Payment.Amount
		}
		if group != "" && g != group {
			continue
		}
		v, err := b.build(ctx, r, viewer)
		if err != nil {
			return nil, err
		}
		out.Items = append(out.Items, v)
	}
	return out, nil
}

func (b *viewBuilder) build(ctx context.Context, r *domain.Reservation, viewer Viewer) (View, error) {
	l, err := memo(ctx, b.listings, r.ListingID, b.src.Listings.FindByID)
	if err != nil {
		return View{}, err
	}
	if l.CycleID == nil {
		return View{}, fmt.Errorf("reservation: listing has no cycle: %w", domain.ErrInvalid)
	}
	c, err := memo(ctx, b.cycles, *l.CycleID, b.src.Cycles.FindByID)
	if err != nil {
		return View{}, err
	}
	f, err := memo(ctx, b.farms, c.FarmID, b.src.Farms.FindByID)
	if err != nil {
		return View{}, err
	}
	dispute, err := hasOpenDispute(ctx, b.src.Reports, r.ID)
	if err != nil {
		return View{}, err
	}
	reviewed, err := b.isReviewed(ctx, f.ID, r.ID)
	if err != nil {
		return View{}, err
	}

	v := View{
		Reservation: r, Listing: l, Cycle: c, Farm: f,
		Group:         domain.GroupOf(r),
		Steps:         domain.EscrowSteps(r),
		AutoReleaseAt: domain.AutoReleaseAt(r, AutoReleaseGrace),
		DisputeOpen:   dispute,
		Reviewed:      reviewed,
	}
	v.Actions = actionsFor(v, viewer)
	return v, nil
}

func (b *viewBuilder) isReviewed(ctx context.Context, farmID, reservationID bson.ObjectID) (bool, error) {
	set, ok := b.reviewed[farmID]
	if !ok {
		reviews, err := b.src.Reviews.ListByFarm(ctx, farmID)
		if err != nil {
			return false, err
		}
		set = make(map[bson.ObjectID]bool, len(reviews))
		for _, rv := range reviews {
			set[rv.ReservationID] = true
		}
		b.reviewed[farmID] = set
	}
	return set[reservationID], nil
}

// actionsFor is the whole "what can I do now" rule set, mirroring the
// checks each action's own usecase enforces.
func actionsFor(v View, viewer Viewer) []Action {
	r := v.Reservation
	payment := domain.PaymentPending
	if r.Payment != nil {
		payment = r.Payment.Status
	}
	var actions []Action
	switch viewer {
	case ViewerBuyer:
		if r.Status == domain.ReservationPending && payment == domain.PaymentPending {
			actions = append(actions, ActionPay, ActionCancel)
		}
		if r.Status == domain.ReservationDelivered && payment == domain.PaymentHeld && !v.DisputeOpen {
			actions = append(actions, ActionConfirm)
		}
		if payment == domain.PaymentHeld && !v.DisputeOpen {
			actions = append(actions, ActionDispute)
		}
		if payment == domain.PaymentReleased && !v.Reviewed {
			actions = append(actions, ActionReview)
		}
		if v.Group == domain.GroupActive && len(v.Farm.Contacts) > 0 {
			actions = append(actions, ActionContact)
		}
	case ViewerFarmer:
		if r.Status == domain.ReservationConfirmed && payment == domain.PaymentHeld && r.DeliveredAt == nil && !v.DisputeOpen {
			actions = append(actions, ActionMarkDelivered)
		}
	}
	return actions
}

func memo[T any](ctx context.Context, cache map[bson.ObjectID]*T, id bson.ObjectID, load func(context.Context, bson.ObjectID) (*T, error)) (*T, error) {
	if v, ok := cache[id]; ok {
		return v, nil
	}
	v, err := load(ctx, id)
	if err != nil {
		return nil, err
	}
	cache[id] = v
	return v, nil
}

// ListMyReservations is the buyer's "Mes réservations": one tab (or all,
// for an empty group) of joined views, plus per-tab counts and the
// amount they currently have held in escrow.
type ListMyReservations struct {
	Reservations ports.ReservationRepository
	Listings     ports.ListingStore
	Cycles       ports.CycleStore
	Farms        ports.FarmRepository
	Reports      ports.ReportRepository
	Reviews      ports.ReviewRepository
}

func (uc *ListMyReservations) Execute(ctx context.Context, buyerID bson.ObjectID, group domain.ReservationGroup) (*ViewList, error) {
	if err := validGroup(group); err != nil {
		return nil, err
	}
	all, err := uc.Reservations.ListByBuyer(ctx, buyerID)
	if err != nil {
		return nil, err
	}
	b := newViewBuilder(viewSources{Listings: uc.Listings, Cycles: uc.Cycles, Farms: uc.Farms, Reports: uc.Reports, Reviews: uc.Reviews})
	return b.list(ctx, all, group, ViewerBuyer)
}

// ListReservationsByFarm is the farmer's side: every reservation on any
// of the farm's listings, owner only.
type ListReservationsByFarm struct {
	Farms        ports.FarmRepository
	Cycles       ports.CycleStore
	Listings     ports.ListingStore
	Reservations ports.ReservationRepository
	Reports      ports.ReportRepository
	Reviews      ports.ReviewRepository
}

type ListReservationsByFarmInput struct {
	FarmID  bson.ObjectID
	OwnerID bson.ObjectID
	Group   domain.ReservationGroup
	// Query narrows the tab to reservations naming it: a reference, a
	// cycle reference or a listing title. The tab counts stay whole.
	Query string
}

// matchesQuery reports whether a view names q (already lower-cased).
func matchesQuery(v View, q string) bool {
	fields := []string{v.Reservation.Reference}
	if v.Cycle != nil {
		fields = append(fields, v.Cycle.Reference)
	}
	if v.Listing != nil {
		fields = append(fields, v.Listing.Title)
	}
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), q) {
			return true
		}
	}
	return false
}

func (uc *ListReservationsByFarm) Execute(ctx context.Context, in ListReservationsByFarmInput) (*ViewList, error) {
	if err := validGroup(in.Group); err != nil {
		return nil, err
	}
	f, err := uc.Farms.FindByID(ctx, in.FarmID)
	if err != nil {
		return nil, err
	}
	if f.OwnerID != in.OwnerID {
		return nil, fmt.Errorf("reservation: not the farm owner: %w", domain.ErrForbidden)
	}

	cycles, err := uc.Cycles.ListByFarm(ctx, in.FarmID)
	if err != nil {
		return nil, err
	}
	listingIDs := make([]bson.ObjectID, 0, len(cycles))
	for _, c := range cycles {
		l, err := uc.Listings.FindByCycleID(ctx, c.ID)
		if errors.Is(err, domain.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		listingIDs = append(listingIDs, l.ID)
	}
	all, err := uc.Reservations.ListByListingIDs(ctx, listingIDs)
	if err != nil {
		return nil, err
	}
	b := newViewBuilder(viewSources{Listings: uc.Listings, Cycles: uc.Cycles, Farms: uc.Farms, Reports: uc.Reports, Reviews: uc.Reviews})
	list, err := b.list(ctx, all, in.Group, ViewerFarmer)
	if err != nil {
		return nil, err
	}
	if q := strings.ToLower(strings.TrimSpace(in.Query)); q != "" {
		kept := list.Items[:0]
		for _, v := range list.Items {
			if matchesQuery(v, q) {
				kept = append(kept, v)
			}
		}
		list.Items = kept
	}
	return list, nil
}

func validGroup(g domain.ReservationGroup) error {
	switch g {
	case "", domain.GroupActive, domain.GroupDone, domain.GroupCancelled:
		return nil
	}
	return fmt.Errorf("reservation: unknown group %q: %w", g, domain.ErrInvalid)
}

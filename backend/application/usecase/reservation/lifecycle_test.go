package reservation

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// fakeListingStore, fakeCycleStore, fakeFarms and fakeReports are the
// fuller-surface doubles MarkDelivered/ConfirmDelivery/AutoRelease need
// (farmOwner/hasOpenDispute walk listing -> cycle -> farm and check
// reports) — kept separate from fakeListings/fakeCycles above, which
// only satisfy ReserveCycle's narrower ports.

type fakeListingStore struct {
	byID map[bson.ObjectID]*domain.Listing
}

func (f *fakeListingStore) Create(context.Context, *domain.Listing) error { return nil }
func (f *fakeListingStore) FindByID(_ context.Context, id bson.ObjectID) (*domain.Listing, error) {
	l, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return l, nil
}
func (f *fakeListingStore) FindByCycleID(context.Context, bson.ObjectID) (*domain.Listing, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeListingStore) Update(context.Context, *domain.Listing) error { return nil }

type fakeCycleStore struct {
	byID map[bson.ObjectID]*domain.Cycle
}

func (f *fakeCycleStore) Create(context.Context, *domain.Cycle) error { return nil }
func (f *fakeCycleStore) FindByID(_ context.Context, id bson.ObjectID) (*domain.Cycle, error) {
	c, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return c, nil
}
func (f *fakeCycleStore) ListByFarm(context.Context, bson.ObjectID) ([]*domain.Cycle, error) {
	return nil, nil
}
func (f *fakeCycleStore) ListByStatus(context.Context, domain.CycleStatus) ([]*domain.Cycle, error) {
	return nil, nil
}
func (f *fakeCycleStore) UpdateStatus(context.Context, bson.ObjectID, domain.CycleStatus) error {
	return nil
}
func (f *fakeCycleStore) Delete(context.Context, bson.ObjectID) error { return nil }

type fakeFarms struct {
	byID map[bson.ObjectID]*domain.Farm
}

func (f *fakeFarms) Create(context.Context, *domain.Farm) error { return nil }
func (f *fakeFarms) FindByID(_ context.Context, id bson.ObjectID) (*domain.Farm, error) {
	farm, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return farm, nil
}
func (f *fakeFarms) ListByOwner(context.Context, bson.ObjectID) ([]*domain.Farm, error) {
	return nil, nil
}
func (f *fakeFarms) Update(context.Context, *domain.Farm) error  { return nil }
func (f *fakeFarms) Delete(context.Context, bson.ObjectID) error { return nil }

type fakeReports struct {
	byTarget map[bson.ObjectID][]*domain.Report
}

func newFakeReports() *fakeReports {
	return &fakeReports{byTarget: map[bson.ObjectID][]*domain.Report{}}
}

func (f *fakeReports) Insert(_ context.Context, r *domain.Report) error {
	f.byTarget[r.TargetID] = append(f.byTarget[r.TargetID], r)
	return nil
}
func (f *fakeReports) FindByID(context.Context, bson.ObjectID) (*domain.Report, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeReports) CountRecentByTarget(_ context.Context, targetID bson.ObjectID, _ time.Time) (int, error) {
	return len(f.byTarget[targetID]), nil
}
func (f *fakeReports) ListByTarget(_ context.Context, targetID bson.ObjectID) ([]*domain.Report, error) {
	return f.byTarget[targetID], nil
}
func (f *fakeReports) List(context.Context, domain.ReportStatus) ([]*domain.Report, error) {
	return nil, nil
}
func (f *fakeReports) Update(_ context.Context, r *domain.Report) error {
	for _, rs := range f.byTarget {
		for i, existing := range rs {
			if existing.ID == r.ID {
				rs[i] = r
			}
		}
	}
	return nil
}

type fakeTrust struct{ calls []bson.ObjectID }

func (f *fakeTrust) Execute(_ context.Context, userID bson.ObjectID) error {
	f.calls = append(f.calls, userID)
	return nil
}

type fakeNotifications struct{ enqueued []*domain.Notification }

func (f *fakeNotifications) Enqueue(_ context.Context, n *domain.Notification) error {
	f.enqueued = append(f.enqueued, n)
	return nil
}
func (f *fakeNotifications) ListPending(context.Context) ([]*domain.Notification, error) {
	return f.enqueued, nil
}
func (f *fakeNotifications) MarkSent(context.Context, bson.ObjectID, time.Time) error { return nil }

// fixture wires a listing -> cycle -> farm chain plus a held reservation
// against it, for the tests below that need farmOwner/hasOpenDispute to
// resolve.
type fixture struct {
	farmOwnerID  bson.ObjectID
	listings     *fakeListingStore
	cycles       *fakeCycleStore
	farms        *fakeFarms
	reservations *fakeReservations
	reports      *fakeReports
	trust        *fakeTrust
	clock        fakeClock
}

func newFixture(now time.Time) *fixture {
	farmID, cycleID, listingID := bson.NewObjectID(), bson.NewObjectID(), bson.NewObjectID()
	ownerID := bson.NewObjectID()
	return &fixture{
		farmOwnerID:  ownerID,
		listings:     &fakeListingStore{byID: map[bson.ObjectID]*domain.Listing{listingID: {ID: listingID, CycleID: &cycleID}}},
		cycles:       &fakeCycleStore{byID: map[bson.ObjectID]*domain.Cycle{cycleID: {ID: cycleID, FarmID: farmID}}},
		farms:        &fakeFarms{byID: map[bson.ObjectID]*domain.Farm{farmID: {ID: farmID, OwnerID: ownerID}}},
		reservations: newFakeReservations(),
		reports:      newFakeReports(),
		trust:        &fakeTrust{},
		clock:        fakeClock{now: now},
	}
}

func (fx *fixture) listingID() bson.ObjectID {
	for id := range fx.listings.byID {
		return id
	}
	return bson.ObjectID{}
}

func TestPayReservation_Execute(t *testing.T) {
	now := time.Now()
	fx := newFixture(now)
	buyerID := bson.NewObjectID()
	res := &domain.Reservation{
		ID: bson.NewObjectID(), ListingID: fx.listingID(), BuyerID: buyerID, TotalAmount: 1000,
		Status: domain.ReservationPending, Payment: &domain.PaymentInfo{Status: domain.PaymentPending, Amount: 1000},
	}
	_ = fx.reservations.Insert(context.Background(), res)
	payments := &fakePayments{}

	uc := &PayReservation{Reservations: fx.reservations, Payments: payments, Clock: fx.clock}
	got, err := uc.Execute(context.Background(), PayReservationInput{ReservationID: res.ID, BuyerID: buyerID})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if got.Status != domain.ReservationConfirmed || !got.Priority || got.Payment.Status != domain.PaymentHeld {
		t.Errorf("got = %+v, want confirmed/priority/held", got)
	}

	// Paying again must fail: no longer pending.
	if _, err := uc.Execute(context.Background(), PayReservationInput{ReservationID: res.ID, BuyerID: buyerID}); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("second pay: error = %v, want domain.ErrConflict", err)
	}

	// A different buyer must be forbidden.
	res2 := &domain.Reservation{
		ID: bson.NewObjectID(), ListingID: fx.listingID(), BuyerID: buyerID, TotalAmount: 500,
		Status: domain.ReservationPending, Payment: &domain.PaymentInfo{Status: domain.PaymentPending, Amount: 500},
	}
	_ = fx.reservations.Insert(context.Background(), res2)
	if _, err := uc.Execute(context.Background(), PayReservationInput{ReservationID: res2.ID, BuyerID: bson.NewObjectID()}); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("wrong buyer: error = %v, want domain.ErrForbidden", err)
	}
}

func TestMarkDeliveredThenConfirmDelivery_ReleasesEscrowAndRecomputesTrust(t *testing.T) {
	now := time.Now()
	fx := newFixture(now)
	buyerID := bson.NewObjectID()
	heldAt := now
	res := &domain.Reservation{
		ID: bson.NewObjectID(), ListingID: fx.listingID(), BuyerID: buyerID, TotalAmount: 2000,
		Status: domain.ReservationConfirmed, Payment: &domain.PaymentInfo{Status: domain.PaymentHeld, Amount: 2000, HeldAt: &heldAt},
	}
	_ = fx.reservations.Insert(context.Background(), res)

	markDelivered := &MarkDelivered{Reservations: fx.reservations, Listings: fx.listings, Cycles: fx.cycles, Farms: fx.farms, Clock: fx.clock}
	delivered, err := markDelivered.Execute(context.Background(), MarkDeliveredInput{ReservationID: res.ID, FarmOwnerID: fx.farmOwnerID})
	if err != nil {
		t.Fatalf("MarkDelivered.Execute() error = %v, want nil", err)
	}
	if delivered.Status != domain.ReservationDelivered || delivered.DeliveredAt == nil {
		t.Fatalf("got = %+v, want delivered with DeliveredAt set", delivered)
	}

	// A non-owner can't mark it delivered.
	if _, err := markDelivered.Execute(context.Background(), MarkDeliveredInput{ReservationID: res.ID, FarmOwnerID: bson.NewObjectID()}); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("wrong owner: error = %v, want domain.ErrForbidden", err)
	}

	confirm := &ConfirmDelivery{
		Reservations: fx.reservations, Reports: fx.reports, Listings: fx.listings, Cycles: fx.cycles,
		Farms: fx.farms, Trust: fx.trust, Clock: fx.clock,
	}
	released, err := confirm.Execute(context.Background(), ConfirmDeliveryInput{ReservationID: res.ID, BuyerID: buyerID})
	if err != nil {
		t.Fatalf("ConfirmDelivery.Execute() error = %v, want nil", err)
	}
	if released.Payment.Status != domain.PaymentReleased || released.Payment.ReleasedAt == nil {
		t.Errorf("Payment = %+v, want released", released.Payment)
	}
	if len(fx.trust.calls) != 1 || fx.trust.calls[0] != fx.farmOwnerID {
		t.Errorf("trust.calls = %v, want one call for %v", fx.trust.calls, fx.farmOwnerID)
	}
}

func TestConfirmDelivery_BlockedByOpenDispute(t *testing.T) {
	now := time.Now()
	fx := newFixture(now)
	buyerID := bson.NewObjectID()
	heldAt := now
	res := &domain.Reservation{
		ID: bson.NewObjectID(), ListingID: fx.listingID(), BuyerID: buyerID, TotalAmount: 2000,
		Status: domain.ReservationDelivered, Payment: &domain.PaymentInfo{Status: domain.PaymentHeld, Amount: 2000, HeldAt: &heldAt},
	}
	_ = fx.reservations.Insert(context.Background(), res)

	dispute := &OpenDispute{Reservations: fx.reservations, Reports: fx.reports, Clock: fx.clock}
	if _, err := dispute.Execute(context.Background(), OpenDisputeInput{ReservationID: res.ID, BuyerID: buyerID, Reason: "1 poussin manquant"}); err != nil {
		t.Fatalf("OpenDispute.Execute() error = %v, want nil", err)
	}

	confirm := &ConfirmDelivery{
		Reservations: fx.reservations, Reports: fx.reports, Listings: fx.listings, Cycles: fx.cycles,
		Farms: fx.farms, Trust: fx.trust, Clock: fx.clock,
	}
	if _, err := confirm.Execute(context.Background(), ConfirmDeliveryInput{ReservationID: res.ID, BuyerID: buyerID}); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("ConfirmDelivery with open dispute: error = %v, want domain.ErrConflict", err)
	}
}

func TestExpireUnpaidReservations_RestoresCapacityAndNotifiesBuyer(t *testing.T) {
	now := time.Now()
	cycleID := bson.NewObjectID()
	listingID := bson.NewObjectID()
	listings := &fakeListings{cycleID: cycleID, unitPrice: 10}
	cycles := &fakeCycles{available: 5}
	reservations := newFakeReservations()
	notifications := &fakeNotifications{}
	buyerID := bson.NewObjectID()

	expired := time.Now().Add(-time.Hour)
	pending := &domain.Reservation{
		ID: bson.NewObjectID(), ListingID: listingID, BuyerID: buyerID, Quantity: 4,
		Status: domain.ReservationPending, PaymentDeadline: &expired,
		Payment: &domain.PaymentInfo{Status: domain.PaymentPending, Amount: 40},
	}
	_ = reservations.Insert(context.Background(), pending)

	uc := &ExpireUnpaidReservations{Reservations: reservations, Listings: listings, Cycles: cycles, Notifications: notifications, Clock: fakeClock{now: now}}
	n, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if n != 1 {
		t.Fatalf("expired count = %d, want 1", n)
	}
	if cycles.available != 9 {
		t.Errorf("cycles.available = %d, want 9 (5 + 4 restored)", cycles.available)
	}
	got := reservations.byID[pending.ID]
	if got.Status != domain.ReservationCancelled || got.Payment.Status != domain.PaymentFailed {
		t.Errorf("got = %+v, want cancelled/failed", got)
	}
	if len(notifications.enqueued) != 1 || notifications.enqueued[0].UserID != buyerID {
		t.Errorf("notifications.enqueued = %v, want one notification to %v", notifications.enqueued, buyerID)
	}
}

func TestAutoReleaseDeliveries_ReleasesUndisputedAndSkipsDisputed(t *testing.T) {
	now := time.Now()
	fx := newFixture(now)
	buyerID := bson.NewObjectID()
	deliveredAt := now.Add(-49 * time.Hour)
	heldAt := now.Add(-72 * time.Hour)

	releasable := &domain.Reservation{
		ID: bson.NewObjectID(), ListingID: fx.listingID(), BuyerID: buyerID, TotalAmount: 1500,
		Status: domain.ReservationDelivered, DeliveredAt: &deliveredAt,
		Payment: &domain.PaymentInfo{Status: domain.PaymentHeld, Amount: 1500, HeldAt: &heldAt},
	}
	disputed := &domain.Reservation{
		ID: bson.NewObjectID(), ListingID: fx.listingID(), BuyerID: buyerID, TotalAmount: 900,
		Status: domain.ReservationDelivered, DeliveredAt: &deliveredAt,
		Payment: &domain.PaymentInfo{Status: domain.PaymentHeld, Amount: 900, HeldAt: &heldAt},
	}
	_ = fx.reservations.Insert(context.Background(), releasable)
	_ = fx.reservations.Insert(context.Background(), disputed)
	_ = fx.reports.Insert(context.Background(), &domain.Report{
		ID: bson.NewObjectID(), TargetID: disputed.ID, TargetType: domain.ReportTargetReservation, Status: domain.ReportOpen,
	})

	uc := &AutoReleaseDeliveries{
		Reservations: fx.reservations, Reports: fx.reports, Listings: fx.listings, Cycles: fx.cycles,
		Farms: fx.farms, Trust: fx.trust, Notifications: &fakeNotifications{}, Clock: fx.clock,
	}
	n, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if n != 1 {
		t.Fatalf("released count = %d, want 1 (the undisputed one only)", n)
	}
	if fx.reservations.byID[releasable.ID].Payment.Status != domain.PaymentReleased {
		t.Error("the undisputed reservation should have been released")
	}
	if fx.reservations.byID[disputed.ID].Payment.Status != domain.PaymentHeld {
		t.Error("the disputed reservation must stay held")
	}
}

var _ ports.Clock = fakeClock{}

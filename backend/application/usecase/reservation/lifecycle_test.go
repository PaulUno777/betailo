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

func (f *fakeCycleStore) AppendHealthEvent(context.Context, bson.ObjectID, domain.HealthEvent, int) error {
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
	res.Quantity = 3
	_ = fx.reservations.Insert(context.Background(), res)
	payments := &fakePayments{}
	cycles := &fakeCycles{available: 5}

	uc := &PayReservation{Reservations: fx.reservations, Listings: &fakeListings{cycleID: bson.NewObjectID()}, Cycles: cycles, Payments: payments, Clock: fx.clock}
	got, err := uc.Execute(context.Background(), PayReservationInput{ReservationID: res.ID, BuyerID: buyerID, PaymentMethod: ports.PaymentMethodCard})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if got.Status != domain.ReservationConfirmed || !got.Priority || got.Payment.Status != domain.PaymentHeld {
		t.Errorf("got = %+v, want confirmed/priority/held", got)
	}
	if cycles.available != 2 {
		t.Errorf("cycles.available = %d, want 2: paying takes the places", cycles.available)
	}

	// Paying again must fail: no longer pending.
	if _, err := uc.Execute(context.Background(), PayReservationInput{ReservationID: res.ID, BuyerID: buyerID, PaymentMethod: ports.PaymentMethodCard}); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("second pay: error = %v, want domain.ErrConflict", err)
	}

	// A different buyer must be forbidden.
	res2 := &domain.Reservation{
		ID: bson.NewObjectID(), ListingID: fx.listingID(), BuyerID: buyerID, TotalAmount: 500,
		Status: domain.ReservationPending, Payment: &domain.PaymentInfo{Status: domain.PaymentPending, Amount: 500},
	}
	_ = fx.reservations.Insert(context.Background(), res2)
	if _, err := uc.Execute(context.Background(), PayReservationInput{ReservationID: res2.ID, BuyerID: bson.NewObjectID(), PaymentMethod: ports.PaymentMethodCard}); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("wrong buyer: error = %v, want domain.ErrForbidden", err)
	}
}

func TestPayReservation_SoldOutAndFailedCapture(t *testing.T) {
	buyerID := bson.NewObjectID()
	newPending := func(reservations *fakeReservations, qty int) *domain.Reservation {
		r := &domain.Reservation{
			ID: bson.NewObjectID(), BuyerID: buyerID, Quantity: qty, TotalAmount: 100,
			Status: domain.ReservationPending, Payment: &domain.PaymentInfo{Status: domain.PaymentPending, Amount: 100},
		}
		_ = reservations.Insert(context.Background(), r)
		return r
	}

	// Paid buyers took the places first: the waiting-list buyer is told
	// it's sold out and charged nothing.
	reservations := newFakeReservations()
	cycles := &fakeCycles{available: 1}
	payments := &fakePayments{}
	uc := &PayReservation{Reservations: reservations, Listings: &fakeListings{}, Cycles: cycles, Payments: payments, Clock: fakeClock{now: time.Now()}}
	res := newPending(reservations, 2)
	_, err := uc.Execute(context.Background(), PayReservationInput{ReservationID: res.ID, BuyerID: buyerID, PaymentMethod: ports.PaymentMethodCard})
	if code, _ := domain.CodeOf(err); !errors.Is(err, domain.ErrInsufficientCapacity) || code != "reservation.sold_out" {
		t.Fatalf("sold out: error = %v (code %q), want ErrInsufficientCapacity / reservation.sold_out", err, code)
	}
	if len(payments.captures) != 0 || res.Status != domain.ReservationPending {
		t.Errorf("captures = %d, status = %s; want no charge and still on the waiting list", len(payments.captures), res.Status)
	}

	// A failed capture gives the places back.
	cycles = &fakeCycles{available: 5}
	uc = &PayReservation{Reservations: reservations, Listings: &fakeListings{}, Cycles: cycles, Payments: &fakePayments{captureErr: errors.New("provider down")}, Clock: fakeClock{now: time.Now()}}
	res = newPending(reservations, 2)
	if _, err := uc.Execute(context.Background(), PayReservationInput{ReservationID: res.ID, BuyerID: buyerID, PaymentMethod: ports.PaymentMethodCard}); err == nil {
		t.Fatal("failed capture: error = nil, want the provider's error")
	}
	if cycles.available != 5 {
		t.Errorf("cycles.available = %d, want 5: a failed payment must not keep the places", cycles.available)
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

func TestCloseUnpaidForCycle_CancelsOnlyTheWaitingList(t *testing.T) {
	now := time.Now()
	listing := &domain.Listing{ID: bson.NewObjectID(), Title: "Poussins"}
	reservations := newFakeReservations()
	notifications := &fakeNotifications{}
	buyerID := bson.NewObjectID()

	unpaid := &domain.Reservation{
		ID: bson.NewObjectID(), ListingID: listing.ID, BuyerID: buyerID, Quantity: 4,
		Status: domain.ReservationPending, Payment: &domain.PaymentInfo{Status: domain.PaymentPending, Amount: 40},
	}
	paid := &domain.Reservation{
		ID: bson.NewObjectID(), ListingID: listing.ID, BuyerID: bson.NewObjectID(), Quantity: 2,
		Status: domain.ReservationConfirmed, Payment: &domain.PaymentInfo{Status: domain.PaymentHeld, Amount: 20},
	}
	_ = reservations.Insert(context.Background(), unpaid)
	_ = reservations.Insert(context.Background(), paid)

	uc := &CloseUnpaidForCycle{Listings: &fakeListings{byCycle: listing}, Reservations: reservations, Notifications: notifications, Clock: fakeClock{now: now}}
	n, err := uc.CloseUnpaid(context.Background(), bson.NewObjectID())
	if err != nil || n != 1 {
		t.Fatalf("CloseUnpaid() = %d, %v; want 1, nil", n, err)
	}
	if got := reservations.byID[unpaid.ID]; got.Status != domain.ReservationCancelled || got.Payment.Status != domain.PaymentFailed {
		t.Errorf("unpaid = %+v, want cancelled/failed", got)
	}
	if got := reservations.byID[paid.ID]; got.Status != domain.ReservationConfirmed {
		t.Errorf("paid status = %s, want untouched", got.Status)
	}
	if len(notifications.enqueued) != 1 || notifications.enqueued[0].UserID != buyerID {
		t.Errorf("notifications = %v, want one to the waiting buyer", notifications.enqueued)
	}

	// Again on the same cycle: nothing left to close.
	if n, err := uc.CloseUnpaid(context.Background(), bson.NewObjectID()); err != nil || n != 0 {
		t.Errorf("second CloseUnpaid() = %d, %v; want 0, nil", n, err)
	}
	// A cycle without a listing has no reservations.
	empty := &CloseUnpaidForCycle{Listings: &fakeListings{}, Reservations: reservations, Clock: fakeClock{now: now}}
	if n, err := empty.CloseUnpaid(context.Background(), bson.NewObjectID()); err != nil || n != 0 {
		t.Errorf("no listing: CloseUnpaid() = %d, %v; want 0, nil", n, err)
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

func TestCancelReservation_OnlyTheBuyersUnpaidReservation(t *testing.T) {
	reservations := newFakeReservations()
	buyerID := bson.NewObjectID()
	pending := &domain.Reservation{
		ID: bson.NewObjectID(), BuyerID: buyerID, Quantity: 4, Status: domain.ReservationPending,
		Payment: &domain.PaymentInfo{Status: domain.PaymentPending},
	}
	_ = reservations.Insert(context.Background(), pending)
	uc := &CancelReservation{Reservations: reservations}

	if _, err := uc.Execute(context.Background(), CancelReservationInput{ReservationID: pending.ID, BuyerID: bson.NewObjectID()}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("someone else's: error = %v, want ErrForbidden", err)
	}
	got, err := uc.Execute(context.Background(), CancelReservationInput{ReservationID: pending.ID, BuyerID: buyerID})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.ReservationCancelled {
		t.Fatalf("status=%s, want cancelled (it held no places, so the cycle is untouched)", got.Status)
	}

	paid := &domain.Reservation{
		ID: bson.NewObjectID(), BuyerID: buyerID, Quantity: 2, Status: domain.ReservationConfirmed,
		Payment: &domain.PaymentInfo{Status: domain.PaymentHeld},
	}
	_ = reservations.Insert(context.Background(), paid)
	if _, err := uc.Execute(context.Background(), CancelReservationInput{ReservationID: paid.ID, BuyerID: buyerID}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("paid: error = %v, want ErrConflict (escrow goes through a dispute)", err)
	}
}

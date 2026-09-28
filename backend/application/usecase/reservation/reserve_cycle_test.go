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

// fakeListings, fakeCycles, fakeReservations, fakePayments, fakeClock
// and fakeTx are in-memory doubles for the ports ReserveCycle depends
// on — the point of defining those ports in application/ports is
// exactly this: the usecase's business rule is tested here without a
// MongoDB instance or a real payment provider.

type fakeListings struct {
	cycleID   bson.ObjectID
	unitPrice float64
	err       error
}

func (f *fakeListings) CycleAndPrice(context.Context, bson.ObjectID) (bson.ObjectID, float64, error) {
	return f.cycleID, f.unitPrice, f.err
}

type fakeCycles struct {
	cycle      *domain.Cycle
	available  int
	decrements []int // records each call's quantity
	increments []int
}

func (f *fakeCycles) FindByID(_ context.Context, id bson.ObjectID) (*domain.Cycle, error) {
	if f.cycle == nil {
		return &domain.Cycle{ID: id, AvailableQuantity: f.available}, nil
	}
	return f.cycle, nil
}

func (f *fakeCycles) DecrementAvailability(_ context.Context, _ bson.ObjectID, quantity int) error {
	if quantity > f.available {
		return domain.ErrInsufficientCapacity
	}
	f.available -= quantity
	f.decrements = append(f.decrements, quantity)
	return nil
}

func (f *fakeCycles) IncrementAvailability(_ context.Context, _ bson.ObjectID, quantity int) error {
	f.available += quantity
	f.increments = append(f.increments, quantity)
	return nil
}

type fakeReservations struct {
	inserted []*domain.Reservation
	updated  []*domain.Reservation
	byID     map[bson.ObjectID]*domain.Reservation
}

func newFakeReservations() *fakeReservations {
	return &fakeReservations{byID: map[bson.ObjectID]*domain.Reservation{}}
}

func (f *fakeReservations) Insert(_ context.Context, r *domain.Reservation) error {
	f.inserted = append(f.inserted, r)
	if f.byID == nil {
		f.byID = map[bson.ObjectID]*domain.Reservation{}
	}
	f.byID[r.ID] = r
	return nil
}

func (f *fakeReservations) FindByID(_ context.Context, id bson.ObjectID) (*domain.Reservation, error) {
	r, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return r, nil
}

func (f *fakeReservations) ListByBuyer(_ context.Context, buyerID bson.ObjectID) ([]*domain.Reservation, error) {
	var out []*domain.Reservation
	for _, r := range f.byID {
		if r.BuyerID == buyerID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeReservations) ListByListingIDs(_ context.Context, listingIDs []bson.ObjectID) ([]*domain.Reservation, error) {
	set := map[bson.ObjectID]bool{}
	for _, id := range listingIDs {
		set[id] = true
	}
	var out []*domain.Reservation
	for _, r := range f.byID {
		if set[r.ListingID] {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeReservations) Update(_ context.Context, r *domain.Reservation) error {
	f.updated = append(f.updated, r)
	if f.byID == nil {
		f.byID = map[bson.ObjectID]*domain.Reservation{}
	}
	f.byID[r.ID] = r
	return nil
}

func (f *fakeReservations) ListUnpaidExpired(_ context.Context, before time.Time) ([]*domain.Reservation, error) {
	var out []*domain.Reservation
	for _, r := range f.byID {
		if r.Status == domain.ReservationPending && r.PaymentDeadline != nil && !r.PaymentDeadline.After(before) {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeReservations) ListDeliveredBefore(_ context.Context, before time.Time) ([]*domain.Reservation, error) {
	var out []*domain.Reservation
	for _, r := range f.byID {
		if r.Status == domain.ReservationDelivered && r.DeliveredAt != nil && !r.DeliveredAt.After(before) &&
			r.Payment != nil && r.Payment.Status == domain.PaymentHeld {
			out = append(out, r)
		}
	}
	return out, nil
}

// fakePayments always succeeds unless captureErr is set.
type fakePayments struct {
	captureErr error
	captures   []float64
}

func (f *fakePayments) Capture(_ context.Context, _ string, amount float64, _ ports.PaymentMethod) (string, error) {
	if f.captureErr != nil {
		return "", f.captureErr
	}
	f.captures = append(f.captures, amount)
	return "fake-ref", nil
}

func (f *fakePayments) Refund(context.Context, string, float64) error { return nil }

// fakeClock returns a fixed instant so deadline math is assertable.
type fakeClock struct{ now time.Time }

func (f fakeClock) Now() time.Time { return f.now }

// fakeTx just runs fn inline — no transaction semantics needed to
// exercise the usecase's orchestration logic itself.
type fakeTx struct{}

func (fakeTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func newReserveCycle(listings *fakeListings, cycles *fakeCycles, reservations *fakeReservations, payments *fakePayments) *ReserveCycle {
	return &ReserveCycle{
		Listings: listings, Cycles: cycles, Reservations: reservations,
		Payments: payments, Tx: fakeTx{}, Clock: fakeClock{now: time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)},
	}
}

func TestReserveCycle_Execute_PayLater_Success(t *testing.T) {
	listings := &fakeListings{cycleID: bson.NewObjectID(), unitPrice: 12.5}
	cycles := &fakeCycles{available: 10}
	reservations := newFakeReservations()
	uc := newReserveCycle(listings, cycles, reservations, &fakePayments{})

	buyerID := bson.NewObjectID()
	out, err := uc.Execute(context.Background(), ReserveCycleInput{
		ListingID: bson.NewObjectID(),
		BuyerID:   buyerID,
		Quantity:  3,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	res := out.Reservation
	if res.Quantity != 3 {
		t.Errorf("Quantity = %d, want 3", res.Quantity)
	}
	if res.UnitPrice != 12.5 {
		t.Errorf("UnitPrice = %v, want 12.5", res.UnitPrice)
	}
	wantFee := 12.5 * 3 * ServiceFeeRate
	if res.ServiceFee != wantFee {
		t.Errorf("ServiceFee = %v, want %v", res.ServiceFee, wantFee)
	}
	if want := 12.5*3 + wantFee; res.TotalAmount != want {
		t.Errorf("TotalAmount = %v, want %v", res.TotalAmount, want)
	}
	if res.Status != domain.ReservationPending {
		t.Errorf("Status = %v, want %v (unpaid)", res.Status, domain.ReservationPending)
	}
	if res.Priority {
		t.Error("Priority = true, want false for an unpaid reservation")
	}
	if res.PaymentDeadline == nil || !res.PaymentDeadline.Equal(uc.Clock.Now().Add(UnpaidReservationGrace)) {
		t.Errorf("PaymentDeadline = %v, want now+%v", res.PaymentDeadline, UnpaidReservationGrace)
	}
	if cycles.available != 7 {
		t.Errorf("cycles.available = %d, want 7 (10 - 3): unpaid reservations still claim real capacity", cycles.available)
	}
	if out.PaymentCaptured {
		t.Error("PaymentCaptured = true, want false: PayNow was not requested")
	}
}

func TestReserveCycle_Execute_PayNow_Success(t *testing.T) {
	listings := &fakeListings{cycleID: bson.NewObjectID(), unitPrice: 10}
	cycles := &fakeCycles{available: 10}
	reservations := newFakeReservations()
	payments := &fakePayments{}
	uc := newReserveCycle(listings, cycles, reservations, payments)

	out, err := uc.Execute(context.Background(), ReserveCycleInput{
		ListingID: bson.NewObjectID(), BuyerID: bson.NewObjectID(), Quantity: 2,
		PayNow: true, PaymentMethod: ports.PaymentMethodMTNMoMo,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	res := out.Reservation
	if !out.PaymentCaptured {
		t.Fatal("PaymentCaptured = false, want true")
	}
	if res.Status != domain.ReservationConfirmed {
		t.Errorf("Status = %v, want %v", res.Status, domain.ReservationConfirmed)
	}
	if !res.Priority {
		t.Error("Priority = false, want true for a paid reservation")
	}
	if res.PaymentDeadline != nil {
		t.Error("PaymentDeadline is set, want nil once paid")
	}
	if res.Payment.Status != domain.PaymentHeld {
		t.Errorf("Payment.Status = %v, want %v", res.Payment.Status, domain.PaymentHeld)
	}
	if len(payments.captures) != 1 {
		t.Fatalf("payments.captures = %d, want 1", len(payments.captures))
	}
	if len(reservations.updated) != 1 {
		t.Errorf("reservations.updated = %d, want 1 (the post-capture persist)", len(reservations.updated))
	}
}

func TestReserveCycle_Execute_PayNow_CaptureFailsKeepsReservationUnpaid(t *testing.T) {
	listings := &fakeListings{cycleID: bson.NewObjectID(), unitPrice: 10}
	cycles := &fakeCycles{available: 10}
	reservations := newFakeReservations()
	payments := &fakePayments{captureErr: errors.New("provider down")}
	uc := newReserveCycle(listings, cycles, reservations, payments)

	out, err := uc.Execute(context.Background(), ReserveCycleInput{
		ListingID: bson.NewObjectID(), BuyerID: bson.NewObjectID(), Quantity: 1, PayNow: true,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil (the reservation itself must still succeed)", err)
	}
	if out.PaymentCaptured {
		t.Error("PaymentCaptured = true, want false")
	}
	if out.Reservation.Status != domain.ReservationPending {
		t.Errorf("Status = %v, want %v: a failed capture must not silently confirm the reservation", out.Reservation.Status, domain.ReservationPending)
	}
}

func TestReserveCycle_Execute_InsufficientCapacity(t *testing.T) {
	listings := &fakeListings{cycleID: bson.NewObjectID(), unitPrice: 10}
	cycles := &fakeCycles{available: 2}
	reservations := newFakeReservations()
	uc := newReserveCycle(listings, cycles, reservations, &fakePayments{})

	_, err := uc.Execute(context.Background(), ReserveCycleInput{
		ListingID: bson.NewObjectID(),
		BuyerID:   bson.NewObjectID(),
		Quantity:  5,
	})
	if !errors.Is(err, domain.ErrInsufficientCapacity) {
		t.Fatalf("Execute() error = %v, want domain.ErrInsufficientCapacity", err)
	}
	if len(reservations.inserted) != 0 {
		t.Errorf("reservations.inserted = %d, want 0 on failure", len(reservations.inserted))
	}
}

func TestReserveCycle_Execute_RejectsNonPositiveQuantity(t *testing.T) {
	uc := newReserveCycle(&fakeListings{}, &fakeCycles{}, newFakeReservations(), &fakePayments{})

	in := ReserveCycleInput{ListingID: bson.NewObjectID(), BuyerID: bson.NewObjectID()}

	in.Quantity = 0
	if _, err := uc.Execute(context.Background(), in); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("Execute() with quantity 0: error = %v, want domain.ErrInvalid", err)
	}

	in.Quantity = -1
	if _, err := uc.Execute(context.Background(), in); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("Execute() with negative quantity: error = %v, want domain.ErrInvalid", err)
	}
}

func TestReserveCycle_Execute_EnforcesMinMaxPerReservation(t *testing.T) {
	cycleID := bson.NewObjectID()
	listings := &fakeListings{cycleID: cycleID, unitPrice: 10}
	cycles := &fakeCycles{cycle: &domain.Cycle{ID: cycleID, MinPerReservation: 5, MaxPerReservation: 20}, available: 100}
	uc := newReserveCycle(listings, cycles, newFakeReservations(), &fakePayments{})

	in := ReserveCycleInput{ListingID: bson.NewObjectID(), BuyerID: bson.NewObjectID()}

	in.Quantity = 2
	if _, err := uc.Execute(context.Background(), in); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("below minimum: error = %v, want domain.ErrInvalid", err)
	}

	in.Quantity = 25
	if _, err := uc.Execute(context.Background(), in); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("above maximum: error = %v, want domain.ErrInvalid", err)
	}

	in.Quantity = 10
	if _, err := uc.Execute(context.Background(), in); err != nil {
		t.Errorf("within bounds: error = %v, want nil", err)
	}
}

func TestReserveCycle_Execute_ListingLookupError(t *testing.T) {
	wantErr := errors.New("listing not found")
	listings := &fakeListings{err: wantErr}
	uc := newReserveCycle(listings, &fakeCycles{}, newFakeReservations(), &fakePayments{})

	_, err := uc.Execute(context.Background(), ReserveCycleInput{
		ListingID: bson.NewObjectID(),
		BuyerID:   bson.NewObjectID(),
		Quantity:  1,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}

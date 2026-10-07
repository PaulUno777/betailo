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

// fakeListings serves one listing for any ID: published by default,
// with status/flagged overridable to exercise the availability rule.
type fakeListings struct {
	// byCycle is what FindByCycleID returns (nil: not found).
	byCycle   *domain.Listing
	cycleID   bson.ObjectID
	unitPrice float64
	err       error
	status    domain.ListingStatus
	flagged   bool
}

func (f *fakeListings) CycleAndPrice(context.Context, bson.ObjectID) (bson.ObjectID, float64, error) {
	return f.cycleID, f.unitPrice, f.err
}

func (f *fakeListings) FindByID(_ context.Context, id bson.ObjectID) (*domain.Listing, error) {
	if f.err != nil {
		return nil, f.err
	}
	status := f.status
	if status == "" {
		status = domain.ListingPublished
	}
	cycleID := f.cycleID
	return &domain.Listing{ID: id, CycleID: &cycleID, UnitPrice: f.unitPrice, Status: status, Flagged: f.flagged}, nil
}

func (f *fakeListings) Create(context.Context, *domain.Listing) error { return nil }
func (f *fakeListings) FindByCycleID(context.Context, bson.ObjectID) (*domain.Listing, error) {
	if f.byCycle == nil {
		return nil, domain.ErrNotFound
	}
	return f.byCycle, nil
}
func (f *fakeListings) Update(context.Context, *domain.Listing) error { return nil }

// testFarm owns every cycle fakeCycles serves; testFarmOwner is the
// farmer behind it.
var (
	testFarmOwner = bson.NewObjectID()
	testFarm      = &domain.Farm{ID: bson.NewObjectID(), OwnerID: testFarmOwner}
)

// fakeSeq counts like the counters collection.
type fakeSeq struct{ n int64 }

func (f *fakeSeq) Next(context.Context, string) (int64, error) {
	f.n++
	return f.n, nil
}

type fakeCycles struct {
	cycle      *domain.Cycle
	available  int
	decrements []int // records each call's quantity
	increments []int
}

func (f *fakeCycles) FindByID(_ context.Context, id bson.ObjectID) (*domain.Cycle, error) {
	if f.cycle == nil {
		return &domain.Cycle{ID: id, FarmID: testFarm.ID, Status: domain.CycleStatusInProgress, Capacity: f.available, AvailableQuantity: f.available}, nil
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

func (f *fakePayments) Capture(_ context.Context, req ports.CaptureRequest) (string, error) {
	amount := req.Amount
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
		Farms: &fakeFarms{byID: map[bson.ObjectID]*domain.Farm{testFarm.ID: testFarm}}, Sequencer: &fakeSeq{},
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
	if cycles.available != 10 || len(cycles.decrements) != 0 {
		t.Errorf("cycles.available = %d, want 10: an unpaid reservation is on the waiting list and takes no places", cycles.available)
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
		PayNow: true, PaymentMethod: ports.PaymentMethodMTNMoMo, PayerPhone: "6 90 12 34 56",
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
	if cycles.available != 8 {
		t.Errorf("cycles.available = %d, want 8: paying takes the places", cycles.available)
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
		ListingID: bson.NewObjectID(), BuyerID: bson.NewObjectID(), Quantity: 1, PayNow: true, PaymentMethod: ports.PaymentMethodCard,
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
	if cycles.available != 10 {
		t.Errorf("cycles.available = %d, want 10: the places go back when the payment fails", cycles.available)
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
		PayNow:    true, PaymentMethod: ports.PaymentMethodCard,
	})
	if code, _ := domain.CodeOf(err); !errors.Is(err, domain.ErrInsufficientCapacity) || code != "reservation.sold_out" {
		t.Fatalf("Execute() error = %v (code %q), want ErrInsufficientCapacity / reservation.sold_out", err, code)
	}
	if len(reservations.inserted) != 0 {
		t.Errorf("reservations.inserted = %d, want 0 on failure", len(reservations.inserted))
	}
}

func TestReserveCycle_Execute_WaitingListAllowedWhenFullButNotBeyondCapacity(t *testing.T) {
	listings := &fakeListings{cycleID: bson.NewObjectID(), unitPrice: 10}
	cycle := &domain.Cycle{ID: bson.NewObjectID(), FarmID: testFarm.ID, Status: domain.CycleStatusInProgress, Capacity: 20, AvailableQuantity: 0}
	reservations := newFakeReservations()
	uc := newReserveCycle(listings, &fakeCycles{cycle: cycle}, reservations, &fakePayments{})

	if _, err := uc.Execute(context.Background(), ReserveCycleInput{ListingID: bson.NewObjectID(), BuyerID: bson.NewObjectID(), Quantity: 5}); err != nil {
		t.Fatalf("join a full cycle's waiting list: error = %v, want nil", err)
	}
	_, err := uc.Execute(context.Background(), ReserveCycleInput{ListingID: bson.NewObjectID(), BuyerID: bson.NewObjectID(), Quantity: 21})
	if code, _ := domain.CodeOf(err); !errors.Is(err, domain.ErrInvalid) || code != "reservation.above_capacity" {
		t.Fatalf("more than the cycle holds: error = %v (code %q), want ErrInvalid / reservation.above_capacity", err, code)
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
	cycles := &fakeCycles{cycle: &domain.Cycle{ID: cycleID, FarmID: testFarm.ID, Status: domain.CycleStatusInProgress, Capacity: 100, MinPerReservation: 5, MaxPerReservation: 20}, available: 100}
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

func TestReserveCycle_Execute_AvailabilityAndOwnershipRules(t *testing.T) {
	tests := []struct {
		name     string
		listings *fakeListings
		cycles   *fakeCycles
		buyer    bson.ObjectID
		want     error
	}{
		{"draft listing", &fakeListings{cycleID: bson.NewObjectID(), unitPrice: 10, status: domain.ListingDraft}, &fakeCycles{available: 10}, bson.NewObjectID(), domain.ErrNotFound},
		{"unpublished listing", &fakeListings{cycleID: bson.NewObjectID(), unitPrice: 10, status: domain.ListingUnpublished}, &fakeCycles{available: 10}, bson.NewObjectID(), domain.ErrNotFound},
		{"flagged listing", &fakeListings{cycleID: bson.NewObjectID(), unitPrice: 10, flagged: true}, &fakeCycles{available: 10}, bson.NewObjectID(), domain.ErrNotFound},
		{"finished cycle", &fakeListings{cycleID: bson.NewObjectID(), unitPrice: 10},
			&fakeCycles{cycle: &domain.Cycle{FarmID: testFarm.ID, Status: domain.CycleStatusDone, AvailableQuantity: 10}, available: 10}, bson.NewObjectID(), domain.ErrConflict},
		{"own listing", &fakeListings{cycleID: bson.NewObjectID(), unitPrice: 10}, &fakeCycles{available: 10}, testFarmOwner, domain.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reservations := newFakeReservations()
			uc := newReserveCycle(tt.listings, tt.cycles, reservations, &fakePayments{})
			_, err := uc.Execute(context.Background(), ReserveCycleInput{ListingID: bson.NewObjectID(), BuyerID: tt.buyer, Quantity: 1})
			if !errors.Is(err, tt.want) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.want)
			}
			if len(tt.cycles.decrements) != 0 {
				t.Error("a refused reservation must not claim any places")
			}
		})
	}
}

func TestReserveCycle_Execute_MobileMoneyNeedsCameroonianMobile(t *testing.T) {
	for _, phone := range []string{"", "222123456", "+33612345678", "12345"} {
		uc := newReserveCycle(&fakeListings{cycleID: bson.NewObjectID(), unitPrice: 10}, &fakeCycles{available: 10}, newFakeReservations(), &fakePayments{})
		_, err := uc.Execute(context.Background(), ReserveCycleInput{
			ListingID: bson.NewObjectID(), BuyerID: bson.NewObjectID(), Quantity: 1,
			PayNow: true, PaymentMethod: ports.PaymentMethodOrangeMoney, PayerPhone: phone,
		})
		if !errors.Is(err, domain.ErrInvalid) {
			t.Errorf("phone %q: error = %v, want ErrInvalid", phone, err)
		}
	}
}

func TestReserveCycle_Execute_AssignsReferenceAndMasksPayer(t *testing.T) {
	uc := newReserveCycle(&fakeListings{cycleID: bson.NewObjectID(), unitPrice: 10}, &fakeCycles{available: 10}, newFakeReservations(), &fakePayments{})
	out, err := uc.Execute(context.Background(), ReserveCycleInput{
		ListingID: bson.NewObjectID(), BuyerID: bson.NewObjectID(), Quantity: 1,
		PayNow: true, PaymentMethod: ports.PaymentMethodMTNMoMo, PayerPhone: "690 12 34 56",
	})
	if err != nil {
		t.Fatal(err)
	}
	res := out.Reservation
	if res.Reference != "RSV-0001" {
		t.Errorf("Reference = %q, want RSV-0001", res.Reference)
	}
	if res.Payment.Method != "mtn_momo" || res.Payment.PayerPhoneMasked != "+237 6•• •• 34 56" || res.Payment.ProviderRef == "" {
		t.Errorf("payment = %+v, want method, masked phone and provider ref recorded", res.Payment)
	}
}

package reservation

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

// fakeListings, fakeCycles, fakeReservations and fakeTx are in-memory
// doubles for the ports ReserveCycle depends on — the point of defining
// those ports in application/ports is exactly this: the usecase's
// business rule is tested here without a MongoDB instance.

type fakeListings struct {
	cycleID   bson.ObjectID
	unitPrice float64
	err       error
}

func (f *fakeListings) CycleAndPrice(context.Context, bson.ObjectID) (bson.ObjectID, float64, error) {
	return f.cycleID, f.unitPrice, f.err
}

type fakeCycles struct {
	available  int
	decrements []int // records each call's quantity
}

func (f *fakeCycles) DecrementAvailability(_ context.Context, _ bson.ObjectID, quantity int) error {
	if quantity > f.available {
		return domain.ErrInsufficientCapacity
	}
	f.available -= quantity
	f.decrements = append(f.decrements, quantity)
	return nil
}

type fakeReservations struct {
	inserted []*domain.Reservation
}

func (f *fakeReservations) Insert(_ context.Context, r *domain.Reservation) error {
	f.inserted = append(f.inserted, r)
	return nil
}

// fakeTx just runs fn inline — no transaction semantics needed to
// exercise the usecase's orchestration logic itself.
type fakeTx struct{}

func (fakeTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func TestReserveCycle_Execute_Success(t *testing.T) {
	listings := &fakeListings{cycleID: bson.NewObjectID(), unitPrice: 12.5}
	cycles := &fakeCycles{available: 10}
	reservations := &fakeReservations{}
	uc := &ReserveCycle{Listings: listings, Cycles: cycles, Reservations: reservations, Tx: fakeTx{}}

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
	if want := 12.5 * 3; res.TotalAmount != want {
		t.Errorf("TotalAmount = %v, want %v", res.TotalAmount, want)
	}
	if res.Status != domain.ReservationConfirmed {
		t.Errorf("Status = %v, want %v", res.Status, domain.ReservationConfirmed)
	}
	if res.BuyerID != buyerID {
		t.Errorf("BuyerID = %v, want %v", res.BuyerID, buyerID)
	}
	if len(reservations.inserted) != 1 {
		t.Fatalf("reservations.inserted = %d, want 1", len(reservations.inserted))
	}
	if cycles.available != 7 {
		t.Errorf("cycles.available = %d, want 7 (10 - 3)", cycles.available)
	}
}

func TestReserveCycle_Execute_InsufficientCapacity(t *testing.T) {
	listings := &fakeListings{cycleID: bson.NewObjectID(), unitPrice: 10}
	cycles := &fakeCycles{available: 2}
	reservations := &fakeReservations{}
	uc := &ReserveCycle{Listings: listings, Cycles: cycles, Reservations: reservations, Tx: fakeTx{}}

	_, err := uc.Execute(context.Background(), ReserveCycleInput{
		ListingID: bson.NewObjectID(),
		BuyerID:   bson.NewObjectID(),
		Quantity:  5,
	})
	if !errors.Is(err, domain.ErrInsufficientCapacity) {
		t.Fatalf("Execute() error = %v, want domain.ErrInsufficientCapacity", err)
	}

	// The failed decrement must not have left a reservation behind — the
	// whole point of running both inside one TxManager.
	if len(reservations.inserted) != 0 {
		t.Errorf("reservations.inserted = %d, want 0 on failure", len(reservations.inserted))
	}
}

func TestReserveCycle_Execute_RejectsNonPositiveQuantity(t *testing.T) {
	uc := &ReserveCycle{Listings: &fakeListings{}, Cycles: &fakeCycles{}, Reservations: &fakeReservations{}, Tx: fakeTx{}}

	in := ReserveCycleInput{ListingID: bson.NewObjectID(), BuyerID: bson.NewObjectID()}

	in.Quantity = 0
	if _, err := uc.Execute(context.Background(), in); err == nil {
		t.Error("Execute() with quantity 0: error = nil, want an error")
	}

	in.Quantity = -1
	if _, err := uc.Execute(context.Background(), in); err == nil {
		t.Error("Execute() with negative quantity: error = nil, want an error")
	}
}

func TestReserveCycle_Execute_ListingLookupError(t *testing.T) {
	wantErr := errors.New("listing not found")
	listings := &fakeListings{err: wantErr}
	uc := &ReserveCycle{Listings: listings, Cycles: &fakeCycles{}, Reservations: &fakeReservations{}, Tx: fakeTx{}}

	_, err := uc.Execute(context.Background(), ReserveCycleInput{
		ListingID: bson.NewObjectID(),
		BuyerID:   bson.NewObjectID(),
		Quantity:  1,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}

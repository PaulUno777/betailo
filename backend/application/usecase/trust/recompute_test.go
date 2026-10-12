package trust

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

type fakeUsers struct {
	byID map[bson.ObjectID]*domain.User
}

func (f *fakeUsers) Create(context.Context, *domain.User) error { return nil }
func (f *fakeUsers) FindByEmail(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeUsers) FindByID(_ context.Context, id bson.ObjectID) (*domain.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}
func (f *fakeUsers) Update(_ context.Context, u *domain.User) error {
	f.byID[u.ID] = u
	return nil
}

func (f *fakeUsers) List(_ context.Context) ([]*domain.User, error) {
	out := make([]*domain.User, 0, len(f.byID))
	for _, u := range f.byID {
		out = append(out, u)
	}
	return out, nil
}

type fakeFarms struct {
	byOwner map[bson.ObjectID][]*domain.Farm
}

func (f *fakeFarms) Create(context.Context, *domain.Farm) error { return nil }
func (f *fakeFarms) FindByID(context.Context, bson.ObjectID) (*domain.Farm, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeFarms) ListByOwner(_ context.Context, ownerID bson.ObjectID) ([]*domain.Farm, error) {
	return f.byOwner[ownerID], nil
}
func (f *fakeFarms) Update(context.Context, *domain.Farm) error  { return nil }
func (f *fakeFarms) Delete(context.Context, bson.ObjectID) error { return nil }

type fakeCycles struct {
	byFarm map[bson.ObjectID][]*domain.Cycle
}

func (f *fakeCycles) Create(context.Context, *domain.Cycle) error { return nil }
func (f *fakeCycles) FindByID(context.Context, bson.ObjectID) (*domain.Cycle, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeCycles) ListByFarm(_ context.Context, farmID bson.ObjectID) ([]*domain.Cycle, error) {
	return f.byFarm[farmID], nil
}
func (f *fakeCycles) ListByStatus(context.Context, domain.CycleStatus) ([]*domain.Cycle, error) {
	return nil, nil
}
func (f *fakeCycles) UpdateStatus(context.Context, bson.ObjectID, domain.CycleStatus) error {
	return nil
}

func (f *fakeCycles) AppendHealthEvent(context.Context, bson.ObjectID, domain.HealthEvent, int) error {
	return nil
}
func (f *fakeCycles) Delete(context.Context, bson.ObjectID) error { return nil }

type fakeListings struct {
	byCycle map[bson.ObjectID]*domain.Listing
}

func (f *fakeListings) Create(context.Context, *domain.Listing) error { return nil }
func (f *fakeListings) FindByID(context.Context, bson.ObjectID) (*domain.Listing, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeListings) ListPublished(context.Context) ([]*domain.Listing, error) { return nil, nil }
func (f *fakeListings) FindByCycleID(_ context.Context, cycleID bson.ObjectID) (*domain.Listing, error) {
	l, ok := f.byCycle[cycleID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return l, nil
}
func (f *fakeListings) Update(context.Context, *domain.Listing) error { return nil }

type fakeReservations struct {
	byListing map[bson.ObjectID][]*domain.Reservation
}

func (f *fakeReservations) Insert(context.Context, *domain.Reservation) error { return nil }
func (f *fakeReservations) FindByID(context.Context, bson.ObjectID) (*domain.Reservation, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeReservations) ListByBuyer(context.Context, bson.ObjectID) ([]*domain.Reservation, error) {
	return nil, nil
}
func (f *fakeReservations) ListByListingIDs(_ context.Context, listingIDs []bson.ObjectID) ([]*domain.Reservation, error) {
	var out []*domain.Reservation
	for _, id := range listingIDs {
		out = append(out, f.byListing[id]...)
	}
	return out, nil
}
func (f *fakeReservations) Update(context.Context, *domain.Reservation) error { return nil }
func (f *fakeReservations) ListDeliveredBefore(context.Context, time.Time) ([]*domain.Reservation, error) {
	return nil, nil
}

type fakeReviews struct {
	byFarm map[bson.ObjectID]ports.ReviewSummary
}

func (f *fakeReviews) Insert(context.Context, *domain.Review) error { return nil }
func (f *fakeReviews) ListByFarm(context.Context, bson.ObjectID) ([]*domain.Review, error) {
	return nil, nil
}
func (f *fakeReviews) Summary(_ context.Context, farmID bson.ObjectID) (ports.ReviewSummary, error) {
	return f.byFarm[farmID], nil
}

type fakeReports struct{ recentCount int }

func (f *fakeReports) Insert(context.Context, *domain.Report) error { return nil }
func (f *fakeReports) FindByID(context.Context, bson.ObjectID) (*domain.Report, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeReports) CountRecentByTarget(context.Context, bson.ObjectID, time.Time) (int, error) {
	return f.recentCount, nil
}
func (f *fakeReports) ListByTarget(context.Context, bson.ObjectID) ([]*domain.Report, error) {
	return nil, nil
}
func (f *fakeReports) List(context.Context, domain.ReportStatus) ([]*domain.Report, error) {
	return nil, nil
}
func (f *fakeReports) Update(context.Context, *domain.Report) error { return nil }

type fakeClock struct{ now time.Time }

func (f fakeClock) Now() time.Time { return f.now }

func TestRecompute_Execute_BaseScoreWithNoSignals(t *testing.T) {
	userID := bson.NewObjectID()
	users := &fakeUsers{byID: map[bson.ObjectID]*domain.User{userID: {ID: userID}}}
	uc := &Recompute{
		Users: users, Farms: &fakeFarms{}, Cycles: &fakeCycles{}, Listings: &fakeListings{},
		Reservations: &fakeReservations{}, Reviews: &fakeReviews{}, Reports: &fakeReports{},
		Clock: fakeClock{now: time.Now()},
	}

	if err := uc.Execute(context.Background(), userID); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if got := users.byID[userID].TrustScore; got != baseScore {
		t.Errorf("TrustScore = %v, want base score %v", got, baseScore)
	}
}

func TestRecompute_Execute_VerifiedIdentityAndSalesAndReviewsRaiseScore(t *testing.T) {
	userID, farmID, cycleID, listingID := bson.NewObjectID(), bson.NewObjectID(), bson.NewObjectID(), bson.NewObjectID()
	users := &fakeUsers{byID: map[bson.ObjectID]*domain.User{userID: {ID: userID, IsVerified: true}}}
	farms := &fakeFarms{byOwner: map[bson.ObjectID][]*domain.Farm{userID: {{ID: farmID, OwnerID: userID}}}}
	cycles := &fakeCycles{byFarm: map[bson.ObjectID][]*domain.Cycle{farmID: {{ID: cycleID, FarmID: farmID}}}}
	listings := &fakeListings{byCycle: map[bson.ObjectID]*domain.Listing{cycleID: {ID: listingID}}}
	released := &domain.Reservation{ListingID: listingID, Payment: &domain.PaymentInfo{Status: domain.PaymentReleased}}
	reservations := &fakeReservations{byListing: map[bson.ObjectID][]*domain.Reservation{listingID: {released, released}}}
	reviews := &fakeReviews{byFarm: map[bson.ObjectID]ports.ReviewSummary{farmID: {Average: 5, Count: 10}}}

	uc := &Recompute{
		Users: users, Farms: farms, Cycles: cycles, Listings: listings,
		Reservations: reservations, Reviews: reviews, Reports: &fakeReports{},
		Clock: fakeClock{now: time.Now()},
	}

	if err := uc.Execute(context.Background(), userID); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	got := users.byID[userID].TrustScore
	// base(50) + verified(15) + sales(2*1.5=3) + reviews(full confidence,
	// (5-3)*10=20) = 88.
	if want := 88.0; got != want {
		t.Errorf("TrustScore = %v, want %v", got, want)
	}
}

func TestRecompute_Execute_ReportsLowerScoreAndClampsAtZero(t *testing.T) {
	userID := bson.NewObjectID()
	users := &fakeUsers{byID: map[bson.ObjectID]*domain.User{userID: {ID: userID}}}
	uc := &Recompute{
		Users: users, Farms: &fakeFarms{}, Cycles: &fakeCycles{}, Listings: &fakeListings{},
		Reservations: &fakeReservations{}, Reviews: &fakeReviews{}, Reports: &fakeReports{recentCount: 10},
		Clock: fakeClock{now: time.Now()},
	}

	if err := uc.Execute(context.Background(), userID); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if got := users.byID[userID].TrustScore; got != 0 {
		t.Errorf("TrustScore = %v, want 0 (clamped)", got)
	}
}

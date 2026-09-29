package listing

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

// fakeFarms, fakeCycles and fakeListings are in-memory doubles for the
// ports these usecases depend on.

type fakeFarms struct {
	byID map[bson.ObjectID]*domain.Farm
}

func newFakeFarms(farms ...*domain.Farm) *fakeFarms {
	f := &fakeFarms{byID: map[bson.ObjectID]*domain.Farm{}}
	for _, farm := range farms {
		f.byID[farm.ID] = farm
	}
	return f
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

type fakeCycles struct {
	byID map[bson.ObjectID]*domain.Cycle
}

func newFakeCycles(cycles ...*domain.Cycle) *fakeCycles {
	c := &fakeCycles{byID: map[bson.ObjectID]*domain.Cycle{}}
	for _, cycle := range cycles {
		c.byID[cycle.ID] = cycle
	}
	return c
}

func (f *fakeCycles) Create(context.Context, *domain.Cycle) error { return nil }
func (f *fakeCycles) FindByID(_ context.Context, id bson.ObjectID) (*domain.Cycle, error) {
	c, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return c, nil
}
func (f *fakeCycles) ListByFarm(context.Context, bson.ObjectID) ([]*domain.Cycle, error) {
	return nil, nil
}
func (f *fakeCycles) ListByStatus(context.Context, domain.CycleStatus) ([]*domain.Cycle, error) {
	return nil, nil
}
func (f *fakeCycles) UpdateStatus(context.Context, bson.ObjectID, domain.CycleStatus) error {
	return nil
}
func (f *fakeCycles) Delete(context.Context, bson.ObjectID) error                     { return nil }
func (f *fakeCycles) DecrementAvailability(context.Context, bson.ObjectID, int) error { return nil }

type fakeListings struct {
	byID map[bson.ObjectID]*domain.Listing
}

func newFakeListings() *fakeListings {
	return &fakeListings{byID: map[bson.ObjectID]*domain.Listing{}}
}

func (f *fakeListings) Create(_ context.Context, l *domain.Listing) error {
	if l.ID.IsZero() {
		l.ID = bson.NewObjectID()
	}
	f.byID[l.ID] = l
	return nil
}

func (f *fakeListings) FindByID(_ context.Context, id bson.ObjectID) (*domain.Listing, error) {
	l, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return l, nil
}

func (f *fakeListings) ListPublished(_ context.Context) ([]*domain.Listing, error) {
	var out []*domain.Listing
	for _, l := range f.byID {
		if l.Status == domain.ListingPublished {
			out = append(out, l)
		}
	}
	return out, nil
}

func (f *fakeListings) FindByCycleID(_ context.Context, cycleID bson.ObjectID) (*domain.Listing, error) {
	for _, l := range f.byID {
		if l.CycleID != nil && *l.CycleID == cycleID {
			return l, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeListings) Update(_ context.Context, l *domain.Listing) error {
	if _, ok := f.byID[l.ID]; !ok {
		return domain.ErrNotFound
	}
	f.byID[l.ID] = l
	return nil
}

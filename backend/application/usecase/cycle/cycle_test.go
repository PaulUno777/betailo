package cycle

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
		if farm.ID.IsZero() {
			farm.ID = bson.NewObjectID()
		}
		f.byID[farm.ID] = farm
	}
	return f
}

func (f *fakeFarms) Create(_ context.Context, farm *domain.Farm) error {
	if farm.ID.IsZero() {
		farm.ID = bson.NewObjectID()
	}
	f.byID[farm.ID] = farm
	return nil
}

func (f *fakeFarms) FindByID(_ context.Context, id bson.ObjectID) (*domain.Farm, error) {
	farm, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return farm, nil
}

func (f *fakeFarms) ListByOwner(_ context.Context, ownerID bson.ObjectID) ([]*domain.Farm, error) {
	var out []*domain.Farm
	for _, farm := range f.byID {
		if farm.OwnerID == ownerID {
			out = append(out, farm)
		}
	}
	return out, nil
}

func (f *fakeFarms) Update(_ context.Context, farm *domain.Farm) error {
	f.byID[farm.ID] = farm
	return nil
}

func (f *fakeFarms) Delete(_ context.Context, id bson.ObjectID) error {
	delete(f.byID, id)
	return nil
}

type fakeCycles struct {
	byID map[bson.ObjectID]*domain.Cycle
}

func newFakeCycles() *fakeCycles {
	return &fakeCycles{byID: map[bson.ObjectID]*domain.Cycle{}}
}

func (f *fakeCycles) Create(_ context.Context, c *domain.Cycle) error {
	if c.ID.IsZero() {
		c.ID = bson.NewObjectID()
	}
	f.byID[c.ID] = c
	return nil
}

func (f *fakeCycles) FindByID(_ context.Context, id bson.ObjectID) (*domain.Cycle, error) {
	c, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return c, nil
}

func (f *fakeCycles) ListByFarm(_ context.Context, farmID bson.ObjectID) ([]*domain.Cycle, error) {
	var out []*domain.Cycle
	for _, c := range f.byID {
		if c.FarmID == farmID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeCycles) ListByStatus(_ context.Context, status domain.CycleStatus) ([]*domain.Cycle, error) {
	var out []*domain.Cycle
	for _, c := range f.byID {
		if c.Status == status {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeCycles) UpdateStatus(_ context.Context, id bson.ObjectID, status domain.CycleStatus) error {
	c, ok := f.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	c.Status = status
	return nil
}

func (f *fakeCycles) Delete(_ context.Context, id bson.ObjectID) error {
	if _, ok := f.byID[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.byID, id)
	return nil
}

func (f *fakeCycles) DecrementAvailability(context.Context, bson.ObjectID, int) error { return nil }

// fakeListings only needs to answer FindByCycleID for DeleteCycle's
// invariant check.
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

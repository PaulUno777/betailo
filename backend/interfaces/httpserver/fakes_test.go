package httpserver

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

// fakeFarmRepo, fakeCycleStore and fakeListingStore are in-memory
// doubles for ports.FarmRepository, ports.CycleStore and
// ports.ListingStore — used the same way fakeUserRepo is in
// router_test.go.

type fakeFarmRepo struct {
	byID map[bson.ObjectID]*domain.Farm
}

func newFakeFarmRepo() *fakeFarmRepo {
	return &fakeFarmRepo{byID: map[bson.ObjectID]*domain.Farm{}}
}

func (f *fakeFarmRepo) Create(_ context.Context, farm *domain.Farm) error {
	if farm.ID.IsZero() {
		farm.ID = bson.NewObjectID()
	}
	f.byID[farm.ID] = farm
	return nil
}

func (f *fakeFarmRepo) FindByID(_ context.Context, id bson.ObjectID) (*domain.Farm, error) {
	farm, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return farm, nil
}

func (f *fakeFarmRepo) ListByOwner(_ context.Context, ownerID bson.ObjectID) ([]*domain.Farm, error) {
	var out []*domain.Farm
	for _, farm := range f.byID {
		if farm.OwnerID == ownerID {
			out = append(out, farm)
		}
	}
	return out, nil
}

func (f *fakeFarmRepo) Update(_ context.Context, farm *domain.Farm) error {
	if _, ok := f.byID[farm.ID]; !ok {
		return domain.ErrNotFound
	}
	f.byID[farm.ID] = farm
	return nil
}

func (f *fakeFarmRepo) Delete(_ context.Context, id bson.ObjectID) error {
	if _, ok := f.byID[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.byID, id)
	return nil
}

type fakeCycleStore struct {
	byID map[bson.ObjectID]*domain.Cycle
}

func newFakeCycleStore() *fakeCycleStore {
	return &fakeCycleStore{byID: map[bson.ObjectID]*domain.Cycle{}}
}

func (f *fakeCycleStore) Create(_ context.Context, c *domain.Cycle) error {
	if c.ID.IsZero() {
		c.ID = bson.NewObjectID()
	}
	f.byID[c.ID] = c
	return nil
}

func (f *fakeCycleStore) FindByID(_ context.Context, id bson.ObjectID) (*domain.Cycle, error) {
	c, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return c, nil
}

func (f *fakeCycleStore) ListByFarm(_ context.Context, farmID bson.ObjectID) ([]*domain.Cycle, error) {
	var out []*domain.Cycle
	for _, c := range f.byID {
		if c.FarmID == farmID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeCycleStore) UpdateStatus(_ context.Context, id bson.ObjectID, status domain.CycleStatus) error {
	c, ok := f.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	c.Status = status
	return nil
}

func (f *fakeCycleStore) Delete(_ context.Context, id bson.ObjectID) error {
	if _, ok := f.byID[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.byID, id)
	return nil
}

type fakeListingStore struct {
	byID map[bson.ObjectID]*domain.Listing
}

func newFakeListingStore() *fakeListingStore {
	return &fakeListingStore{byID: map[bson.ObjectID]*domain.Listing{}}
}

func (f *fakeListingStore) Create(_ context.Context, l *domain.Listing) error {
	if l.ID.IsZero() {
		l.ID = bson.NewObjectID()
	}
	f.byID[l.ID] = l
	return nil
}

func (f *fakeListingStore) FindByID(_ context.Context, id bson.ObjectID) (*domain.Listing, error) {
	l, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return l, nil
}

func (f *fakeListingStore) ListPublished(_ context.Context) ([]*domain.Listing, error) {
	var out []*domain.Listing
	for _, l := range f.byID {
		if l.Status == domain.ListingPublished {
			out = append(out, l)
		}
	}
	return out, nil
}

func (f *fakeListingStore) FindByCycleID(_ context.Context, cycleID bson.ObjectID) (*domain.Listing, error) {
	for _, l := range f.byID {
		if l.CycleID != nil && *l.CycleID == cycleID {
			return l, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeListingStore) Update(_ context.Context, l *domain.Listing) error {
	if _, ok := f.byID[l.ID]; !ok {
		return domain.ErrNotFound
	}
	f.byID[l.ID] = l
	return nil
}

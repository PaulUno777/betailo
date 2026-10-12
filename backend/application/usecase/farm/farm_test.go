package farm

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

// fakeFarms and fakeCycleStore are in-memory doubles for the ports
// these usecases depend on.

type fakeFarms struct {
	byID map[bson.ObjectID]*domain.Farm
}

func newFakeFarms() *fakeFarms {
	return &fakeFarms{byID: map[bson.ObjectID]*domain.Farm{}}
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
	if _, ok := f.byID[farm.ID]; !ok {
		return domain.ErrNotFound
	}
	f.byID[farm.ID] = farm
	return nil
}

func (f *fakeFarms) Delete(_ context.Context, id bson.ObjectID) error {
	if _, ok := f.byID[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.byID, id)
	return nil
}

// fakeCycleStore only needs to answer ListByFarm for DeleteFarm's
// invariant check — the other methods exist to satisfy ports.CycleStore
// and are never called by anything in this package's tests.
type fakeCycleStore struct {
	byFarm map[bson.ObjectID][]*domain.Cycle
}

func (f *fakeCycleStore) Create(context.Context, *domain.Cycle) error { return nil }
func (f *fakeCycleStore) FindByID(context.Context, bson.ObjectID) (*domain.Cycle, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeCycleStore) ListByFarm(_ context.Context, farmID bson.ObjectID) ([]*domain.Cycle, error) {
	return f.byFarm[farmID], nil
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

type fakeClock struct{ now time.Time }

func (c fakeClock) Now() time.Time { return c.now }

type fakeTrust struct{ calls []bson.ObjectID }

func (f *fakeTrust) Execute(_ context.Context, userID bson.ObjectID) error {
	f.calls = append(f.calls, userID)
	return nil
}

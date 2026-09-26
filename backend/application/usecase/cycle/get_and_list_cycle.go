package cycle

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// GetCycle is the usecase: look up a cycle by ID. Like GetFarm, reading
// needs no ownership check — a buyer browsing a listing needs to see
// its cycle's details.
type GetCycle struct {
	Cycles ports.CycleStore
}

func (uc *GetCycle) Execute(ctx context.Context, id bson.ObjectID) (*domain.Cycle, error) {
	return uc.Cycles.FindByID(ctx, id)
}

// ListCyclesByFarm is the usecase: list every cycle belonging to a
// farm.
type ListCyclesByFarm struct {
	Cycles ports.CycleStore
}

func (uc *ListCyclesByFarm) Execute(ctx context.Context, farmID bson.ObjectID) ([]*domain.Cycle, error) {
	return uc.Cycles.ListByFarm(ctx, farmID)
}

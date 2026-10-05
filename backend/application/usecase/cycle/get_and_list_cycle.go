package cycle

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// GetCycle is the usecase: look up a cycle by ID, with its timeline.
// Reading needs no ownership check: a buyer browsing a listing needs to
// see its cycle's details.
type GetCycle struct {
	Cycles ports.CycleStore
	Clock  ports.Clock
}

// CycleView is a cycle plus its timeline as of the read, built
// server-side so every client shows the same step as "current".
type CycleView struct {
	Cycle    *domain.Cycle
	Timeline []domain.TimelineStep
}

func (uc *GetCycle) Execute(ctx context.Context, id bson.ObjectID) (*CycleView, error) {
	c, err := uc.Cycles.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &CycleView{Cycle: c, Timeline: domain.CycleTimeline(c, uc.Clock.Now())}, nil
}

// ListCyclesByFarm is the usecase: list every cycle belonging to a
// farm.
type ListCyclesByFarm struct {
	Cycles ports.CycleStore
}

func (uc *ListCyclesByFarm) Execute(ctx context.Context, farmID bson.ObjectID) ([]*domain.Cycle, error) {
	return uc.Cycles.ListByFarm(ctx, farmID)
}

package farm

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// DeleteFarm is the usecase: remove a farm the caller owns, provided it
// has no cycles — deleting a farm out from under an existing cycle
// would orphan that cycle's FarmID reference.
type DeleteFarm struct {
	Farms  ports.FarmRepository
	Cycles ports.CycleStore
}

// Execute returns domain.ErrNotFound if the farm doesn't exist,
// domain.ErrForbidden if ownerID isn't the farm's actual owner, and
// domain.ErrConflict if the farm still has cycles.
func (uc *DeleteFarm) Execute(ctx context.Context, farmID, ownerID bson.ObjectID) error {
	f, err := uc.Farms.FindByID(ctx, farmID)
	if err != nil {
		return err
	}
	if f.OwnerID != ownerID {
		return fmt.Errorf("farm: not the owner: %w", domain.ErrForbidden)
	}

	cycles, err := uc.Cycles.ListByFarm(ctx, farmID)
	if err != nil {
		return err
	}
	if len(cycles) > 0 {
		return fmt.Errorf("farm: has %d existing cycle(s): %w", len(cycles), domain.ErrConflict)
	}

	return uc.Farms.Delete(ctx, farmID)
}

package farm

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// GetFarm is the usecase: look up a farm by ID. Reading a farm needs no
// ownership check — farms are public (a buyer browsing listings should
// be able to see which farm they come from); only writes are
// owner-restricted.
type GetFarm struct {
	Farms ports.FarmRepository
}

// Execute returns domain.ErrNotFound if no farm has this ID.
func (uc *GetFarm) Execute(ctx context.Context, id bson.ObjectID) (*domain.Farm, error) {
	return uc.Farms.FindByID(ctx, id)
}

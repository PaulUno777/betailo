package farm

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// ListMyFarms is the usecase: list every farm owned by the
// authenticated user.
type ListMyFarms struct {
	Farms ports.FarmRepository
}

func (uc *ListMyFarms) Execute(ctx context.Context, ownerID bson.ObjectID) ([]*domain.Farm, error) {
	return uc.Farms.ListByOwner(ctx, ownerID)
}

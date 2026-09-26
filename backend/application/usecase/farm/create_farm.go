// Package farm implements CRUD for a farmer's operation — Phase 2 of
// the roadmap ("Betailo — Plan de développement"). Like
// application/usecase/auth and .../reservation, every usecase here
// depends only on application/ports, never on mongo-driver directly.
package farm

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// CreateFarm is the usecase: validate input and persist a new farm
// owned by the authenticated user.
type CreateFarm struct {
	Farms ports.FarmRepository
	Clock ports.Clock
}

// CreateFarmInput is what a caller (the HTTP handler) supplies. OwnerID
// comes from the authenticated user's token, never from the request
// body — a farmer can only ever create a farm for themselves.
type CreateFarmInput struct {
	OwnerID  bson.ObjectID
	Name     string
	Location string
	Species  []string
}

// CreateFarmOutput wraps the usecase's result.
type CreateFarmOutput struct {
	Farm *domain.Farm
}

// Execute validates in, then creates the farm.
func (uc *CreateFarm) Execute(ctx context.Context, in CreateFarmInput) (*CreateFarmOutput, error) {
	name := strings.TrimSpace(in.Name)
	location := strings.TrimSpace(in.Location)
	if name == "" {
		return nil, fmt.Errorf("farm: name is required: %w", domain.ErrInvalid)
	}
	if location == "" {
		return nil, fmt.Errorf("farm: location is required: %w", domain.ErrInvalid)
	}

	f := &domain.Farm{
		OwnerID:   in.OwnerID,
		Name:      name,
		Location:  location,
		Species:   in.Species,
		CreatedAt: uc.Clock.Now(),
	}

	if err := uc.Farms.Create(ctx, f); err != nil {
		return nil, err
	}

	return &CreateFarmOutput{Farm: f}, nil
}

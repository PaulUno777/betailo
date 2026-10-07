package farm

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// UpdateFarm is the usecase: update a farm's editable fields, after
// confirming the caller owns it.
type UpdateFarm struct {
	Farms ports.FarmRepository
}

// UpdateFarmInput is what a caller (the HTTP handler) supplies. OwnerID
// comes from the authenticated user's token and is compared against
// the farm's actual owner — never trust a client-supplied owner.
type UpdateFarmInput struct {
	FarmID   bson.ObjectID
	OwnerID  bson.ObjectID
	Name     string
	Location string
	Species  []string

	Description  string
	ContactPhone string
}

// Execute returns domain.ErrNotFound if the farm doesn't exist and
// domain.ErrForbidden if OwnerID isn't the farm's actual owner.
func (uc *UpdateFarm) Execute(ctx context.Context, in UpdateFarmInput) (*domain.Farm, error) {
	f, err := uc.Farms.FindByID(ctx, in.FarmID)
	if err != nil {
		return nil, err
	}
	if f.OwnerID != in.OwnerID {
		return nil, fmt.Errorf("farm: not the owner: %w", domain.ErrForbidden)
	}

	name := strings.TrimSpace(in.Name)
	location := strings.TrimSpace(in.Location)
	if name == "" {
		return nil, domain.Coded("farm.name_required", fmt.Errorf("farm: name is required: %w", domain.ErrInvalid))
	}
	if location == "" {
		return nil, domain.Coded("farm.location_required", fmt.Errorf("farm: location is required: %w", domain.ErrInvalid))
	}

	description, phone, err := normalizeProfileFields(in.Description, in.ContactPhone)
	if err != nil {
		return nil, err
	}

	f.Name = name
	f.Location = location
	f.Species = in.Species
	f.Description = description
	f.ContactPhone = phone

	if err := uc.Farms.Update(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

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
// owned by the authenticated user. Any personal account can open one,
// up to its limit; the farm starts as a draft and can prepare cycles
// and listings, but sells only once staff approved it (see
// SubmitFarmForReview and ReviewFarm).
type CreateFarm struct {
	Users ports.UserRepository
	Farms ports.FarmRepository
	// Catalog says which countries and regions a farm can be in.
	Catalog ports.CatalogSource
	// MaxFarms is the platform's default number of farms per account
	// (MAX_FARMS_PER_ACCOUNT); an account's own User.FarmLimit, set by
	// staff, takes precedence.
	MaxFarms int
	// MaxContacts is how many contact numbers a farm may list
	// (MAX_FARM_CONTACTS); 0 means domain.DefaultMaxFarmContacts.
	MaxContacts int
	// Trust gives the new farmer their baseline trust score straight
	// away. Without it, the score reads 0/100 until the first sale,
	// review or report.
	Trust ports.TrustRecomputer
	Clock ports.Clock
	// Sequencer numbers the farm ("FRM-0212"); nil leaves it without a
	// reference.
	Sequencer ports.Sequencer
}

// CreateFarmInput is what a caller (the HTTP handler) supplies. OwnerID
// comes from the authenticated user's token, never from the request
// body — a farmer can only ever create a farm for themselves.
type CreateFarmInput struct {
	OwnerID  bson.ObjectID
	Name     string
	Location string
	Country  string
	Region   string
	Species  []string

	Description string
	Contacts    []domain.FarmContact
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
		return nil, domain.Coded("farm.name_required", fmt.Errorf("farm: name is required: %w", domain.ErrInvalid))
	}
	if location == "" {
		return nil, domain.Coded("farm.location_required", fmt.Errorf("farm: location is required: %w", domain.ErrInvalid))
	}

	where, err := normalizePlace(ctx, uc.Catalog, in.Country, in.Region, location)
	if err != nil {
		return nil, err
	}
	description, contacts, err := normalizeProfileFields(in.Description, in.Contacts, where.Country.CallingCode, ContactLimit(uc.MaxContacts))
	if err != nil {
		return nil, err
	}

	species, err := normalizeSpecies(ctx, uc.Catalog, in.Species)
	if err != nil {
		return nil, err
	}

	owner, err := uc.Users.FindByID(ctx, in.OwnerID)
	if err != nil {
		return nil, err
	}
	if owner.StaffRole.IsStaff() {
		return nil, domain.Coded("farm.staff_cannot_sell", fmt.Errorf("farm: staff accounts don't sell: %w", domain.ErrForbidden))
	}
	owned, err := uc.Farms.ListByOwner(ctx, in.OwnerID)
	if err != nil {
		return nil, err
	}
	if limit := FarmLimitFor(owner, uc.MaxFarms); len(owned) >= limit {
		return nil, domain.CodedWith("farm.limit_reached", map[string]any{"max": limit}, fmt.Errorf("farm: this account may own at most %d farm(s): %w", limit, domain.ErrConflict))
	}

	f := &domain.Farm{
		Status:    domain.FarmDraft,
		OwnerID:   in.OwnerID,
		Name:      name,
		Location:  where.Town,
		Country:   where.Country.Code,
		Region:    where.Region,
		Species:   species,
		CreatedAt: uc.Clock.Now(),

		Description: description,
		Contacts:    contacts,
	}

	if uc.Sequencer != nil {
		ref, err := uc.Sequencer.NextReference(ctx, domain.FarmRefPrefix)
		if err != nil {
			return nil, err
		}
		f.Reference = ref
	}

	if err := uc.Farms.Create(ctx, f); err != nil {
		return nil, err
	}
	if err := uc.Trust.Execute(ctx, in.OwnerID); err != nil {
		return nil, err
	}

	return &CreateFarmOutput{Farm: f}, nil
}

// FarmLimitFor is how many farms an account may own: what staff set for
// it, else the platform default, else domain.DefaultMaxFarmsPerAccount.
func FarmLimitFor(u *domain.User, platformDefault int) int {
	if u.FarmLimit > 0 {
		return u.FarmLimit
	}
	if platformDefault > 0 {
		return platformDefault
	}
	return domain.DefaultMaxFarmsPerAccount
}

// ContactLimit is how many contact numbers a farm may list: the
// platform's setting, else domain.DefaultMaxFarmContacts.
func ContactLimit(platform int) int {
	if platform > 0 {
		return platform
	}
	return domain.DefaultMaxFarmContacts
}

// SellerAccess is what an account can do as a seller, for the client to
// draw its menus from: the farms it owns and whether it may open
// another. The same rule CreateFarm enforces, read here.
type SellerAccess struct {
	Users       ports.UserRepository
	Farms       ports.FarmRepository
	MaxFarms    int
	MaxContacts int
}

type SellerAccessOutput struct {
	Farms     []*domain.Farm
	FarmLimit int
	CanAdd    bool
	// ContactLimit is how many contact numbers each farm may list.
	ContactLimit int
}

func (uc *SellerAccess) Execute(ctx context.Context, userID bson.ObjectID) (*SellerAccessOutput, error) {
	u, err := uc.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	farms, err := uc.Farms.ListByOwner(ctx, userID)
	if err != nil {
		return nil, err
	}
	limit := FarmLimitFor(u, uc.MaxFarms)
	return &SellerAccessOutput{Farms: farms, FarmLimit: limit, CanAdd: !u.StaffRole.IsStaff() && len(farms) < limit, ContactLimit: ContactLimit(uc.MaxContacts)}, nil
}

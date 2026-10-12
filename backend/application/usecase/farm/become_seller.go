package farm

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// BecomeSeller turns a buyer into a farmer: their farm is created and
// their role switched in one transaction, so there's never a farmer
// without a farm or a farm owned by a buyer. The client refreshes its
// access token afterwards to pick up the new role (refresh re-reads the
// user).
type BecomeSeller struct {
	Users ports.UserRepository
	Farms ports.FarmRepository
	Trust ports.TrustRecomputer
	Tx    ports.TxManager
	Clock ports.Clock
}

type BecomeSellerInput struct {
	UserID       bson.ObjectID
	FarmName     string
	Location     string
	Species      []string
	Description  string
	ContactPhone string
}

// Execute returns domain.ErrConflict for someone already selling and
// domain.ErrForbidden for an admin account.
func (uc *BecomeSeller) Execute(ctx context.Context, in BecomeSellerInput) (*domain.Farm, error) {
	name, location := strings.TrimSpace(in.FarmName), strings.TrimSpace(in.Location)
	if name == "" || location == "" {
		return nil, domain.Coded("farm.name_location_required", fmt.Errorf("farm: name and location are required: %w", domain.ErrInvalid))
	}
	description, phone, err := normalizeProfileFields(in.Description, in.ContactPhone)
	if err != nil {
		return nil, err
	}

	u, err := uc.Users.FindByID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	switch u.Role {
	case domain.RoleFarmer:
		return nil, domain.Coded("farm.already_seller", fmt.Errorf("farm: already a seller: %w", domain.ErrConflict))
	case domain.RoleAdmin:
		return nil, domain.Coded("farm.admin_cannot_sell", fmt.Errorf("farm: admin accounts don't sell: %w", domain.ErrForbidden))
	}

	f := &domain.Farm{
		OwnerID: u.ID, Name: name, Location: location, Species: in.Species, CreatedAt: uc.Clock.Now(),
		Description: description, ContactPhone: phone,
	}
	err = uc.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := uc.Farms.Create(ctx, f); err != nil {
			return err
		}
		u.Role = domain.RoleFarmer
		return uc.Users.Update(ctx, u)
	})
	if err != nil {
		return nil, err
	}
	_ = uc.Trust.Execute(ctx, u.ID)
	return f, nil
}

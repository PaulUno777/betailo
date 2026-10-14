// Package admin implements the root-only console: platform-wide
// oversight (users, cycles, listings, escrow, reports) that every other
// module deliberately scopes to "the caller's own farm" or "public
// data" — this is the one place that reads and mutates across every
// farm at once, gated by RequireRole(domain.RoleAdmin) at the HTTP
// layer, never by a check inside these usecases themselves.
package admin

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// ListUsers backs the admin console's "Utilisateurs" page.
type ListUsers struct {
	Users ports.UserRepository
}

func (uc *ListUsers) Execute(ctx context.Context) ([]*domain.User, error) {
	return uc.Users.List(ctx)
}

// MaxFarmLimit caps what staff can grant one account.
const MaxFarmLimit = 50

// SetFarmLimit lets staff raise (or reset) how many farms one account
// may own. 0 puts the account back on the platform default
// (MAX_FARMS_PER_ACCOUNT). Lowering it below what the account already
// owns removes nothing: it only stops new farms.
type SetFarmLimit struct {
	Users ports.UserRepository
}

func (uc *SetFarmLimit) Execute(ctx context.Context, userID bson.ObjectID, limit int) (*domain.User, error) {
	if limit < 0 || limit > MaxFarmLimit {
		return nil, domain.CodedWith("admin.farm_limit_range", map[string]any{"max": MaxFarmLimit}, fmt.Errorf("admin: farm limit must be between 0 and %d: %w", MaxFarmLimit, domain.ErrInvalid))
	}
	u, err := uc.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	u.FarmLimit = limit
	if err := uc.Users.Update(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// SuspendUser is "Suspendre le compte" — blocks the account from
// creating new farms/cycles/listings/reservations/reviews/reports/forum
// posts (see domain.User.Suspended's doc comment); it does not affect
// reservations or escrow already in flight.
type SuspendUser struct {
	Users ports.UserRepository
	// Sessions, when set, is told so the suspension applies on the
	// user's very next request instead of after the gate's cache delay.
	Sessions SessionForgetter
}

// SessionForgetter drops what the auth layer cached about an account.
type SessionForgetter interface {
	Forget(userID bson.ObjectID)
}

func (uc *SuspendUser) Execute(ctx context.Context, userID bson.ObjectID) (*domain.User, error) {
	u, err := uc.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	u.Suspended = true
	if err := uc.Users.Update(ctx, u); err != nil {
		return nil, err
	}
	if uc.Sessions != nil {
		uc.Sessions.Forget(u.ID)
	}
	return u, nil
}

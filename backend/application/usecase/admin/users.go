// Package admin implements the root-only console: platform-wide
// oversight (users, cycles, listings, escrow, reports) that every other
// module deliberately scopes to "the caller's own farm" or "public
// data" — this is the one place that reads and mutates across every
// farm at once, gated by RequireRole(domain.RoleAdmin) at the HTTP
// layer, never by a check inside these usecases themselves.
package admin

import (
	"context"

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

// VerifyUser marks a user's identity verified — "Demander une
// vérification d'identité" resolved positively — and recomputes their
// trust score, since IsVerified directly feeds it.
type VerifyUser struct {
	Users ports.UserRepository
	Trust ports.TrustRecomputer
}

func (uc *VerifyUser) Execute(ctx context.Context, userID bson.ObjectID) (*domain.User, error) {
	u, err := uc.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	u.IsVerified = true
	if err := uc.Users.Update(ctx, u); err != nil {
		return nil, err
	}
	_ = uc.Trust.Execute(ctx, userID)
	return u, nil
}

// SuspendUser is "Suspendre le compte" — blocks the account from
// creating new farms/cycles/listings/reservations/reviews/reports/forum
// posts (see domain.User.Suspended's doc comment); it does not affect
// reservations or escrow already in flight.
type SuspendUser struct {
	Users ports.UserRepository
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
	return u, nil
}

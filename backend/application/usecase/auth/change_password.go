package auth

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// ChangePassword replaces the user's password after checking the
// current one, and ends every session they have: whoever else might
// know the old password is signed out everywhere. The HTTP layer then
// signs this device back in with the new password.
type ChangePassword struct {
	Users         ports.UserRepository
	Hasher        ports.PasswordHasher
	RefreshTokens ports.RefreshTokenRepository
	Clock         ports.Clock
}

type ChangePasswordInput struct {
	UserID  bson.ObjectID
	Current string
	New     string
}

// Execute returns domain.ErrUnauthorized for a wrong current password
// and domain.ErrInvalid for a too-short or unchanged new one.
func (uc *ChangePassword) Execute(ctx context.Context, in ChangePasswordInput) (*domain.User, error) {
	if len(in.New) < minPasswordLength {
		return nil, fmt.Errorf("auth: password must be at least %d characters: %w", minPasswordLength, domain.ErrInvalid)
	}
	if in.New == in.Current {
		return nil, fmt.Errorf("auth: the new password must differ from the current one: %w", domain.ErrInvalid)
	}
	u, err := uc.Users.FindByID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	if err := uc.Hasher.Compare(u.PasswordHash, in.Current); err != nil {
		return nil, fmt.Errorf("auth: current password is wrong: %w", domain.ErrUnauthorized)
	}
	hash, err := uc.Hasher.Hash(in.New)
	if err != nil {
		return nil, err
	}
	u.PasswordHash = hash
	if err := uc.Users.Update(ctx, u); err != nil {
		return nil, err
	}
	if err := uc.RefreshTokens.RevokeAllForUser(ctx, u.ID, "password_changed", uc.Clock.Now()); err != nil {
		return nil, err
	}
	return u, nil
}

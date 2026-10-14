package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// EnsureSuperAdmin makes sure the platform has its first super admin:
// nobody can register into the staff, so the very first one comes from
// the deployment's configuration (SUPER_ADMIN_EMAIL and
// SUPER_ADMIN_PASSWORD), at startup. Idempotent: an existing account
// with that email is promoted and keeps its password; otherwise the
// account is created with the configured one.
type EnsureSuperAdmin struct {
	Users  ports.UserRepository
	Hasher ports.PasswordHasher
	Clock  ports.Clock
}

// Execute reports whether it created the account (false: it existed).
func (uc *EnsureSuperAdmin) Execute(ctx context.Context, rawEmail, password string) (created bool, err error) {
	email := normalizeEmail(rawEmail)
	if email == "" {
		return false, fmt.Errorf("auth: super admin email is empty: %w", domain.ErrInvalid)
	}
	u, err := uc.Users.FindByEmail(ctx, email)
	if err == nil {
		if u.StaffRole == domain.StaffSuperAdmin {
			return false, nil
		}
		u.StaffRole = domain.StaffSuperAdmin
		return false, uc.Users.Update(ctx, u)
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return false, err
	}
	if len(password) < minPasswordLength {
		return false, fmt.Errorf("auth: super admin password must be at least %d characters: %w", minPasswordLength, domain.ErrInvalid)
	}
	hash, err := uc.Hasher.Hash(password)
	if err != nil {
		return false, err
	}
	err = uc.Users.Create(ctx, &domain.User{
		Email: email, PasswordHash: hash, StaffRole: domain.StaffSuperAdmin, DisplayName: "Betailo", CreatedAt: uc.Clock.Now(),
	})
	return err == nil, err
}

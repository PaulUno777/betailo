package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// EnsureRootAdmin gives the platform its root account on day zero.
// Nobody can register into the staff and only a super admin can grant a
// role, so the very first one has to come from outside the API: the
// deployment's configuration (SUPER_ADMIN_EMAIL, SUPER_ADMIN_PASSWORD),
// read at every startup.
//
// The root account is a super admin that the API can't demote or
// suspend (see admin.SetStaffRole and admin.SuspendUser): whatever
// happens in the back office, whoever holds the deployment keeps a way
// in. There is exactly one: pointing the configuration at another
// email moves the mark, and the former root stays an ordinary super
// admin, to be revoked from the back office if it should be.
type EnsureRootAdmin struct {
	Users  ports.UserRepository
	Hasher ports.PasswordHasher
	Clock  ports.Clock
}

// RootAdminOutcome says what Execute had to do.
type RootAdminOutcome string

const (
	// RootAdminUnchanged: the configured account already was the root.
	RootAdminUnchanged RootAdminOutcome = "unchanged"
	// RootAdminCreated: no account had that email; one was created
	// with the configured password.
	RootAdminCreated RootAdminOutcome = "created"
	// RootAdminPromoted: an existing account was made root. It keeps
	// the password it had.
	RootAdminPromoted RootAdminOutcome = "promoted"
)

// Execute is idempotent. An existing account keeps its password: the
// configured one only ever sets the password of an account this call
// creates, so rotating the variable never locks the root out and never
// resets a password behind its owner's back.
func (uc *EnsureRootAdmin) Execute(ctx context.Context, rawEmail, password string) (RootAdminOutcome, error) {
	email := normalizeEmail(rawEmail)
	if email == "" || !strings.Contains(email, "@") {
		return "", fmt.Errorf("auth: root admin email %q is not an email address: %w", rawEmail, domain.ErrInvalid)
	}

	u, err := uc.Users.FindByEmail(ctx, email)
	switch {
	case err == nil:
		if u.Root && u.StaffRole == domain.StaffSuperAdmin && !u.Suspended {
			return RootAdminUnchanged, uc.unmarkOthers(ctx, u)
		}
		u.Root, u.StaffRole, u.Suspended = true, domain.StaffSuperAdmin, false
		if err := uc.Users.Update(ctx, u); err != nil {
			return "", err
		}
		return RootAdminPromoted, uc.unmarkOthers(ctx, u)
	case !errors.Is(err, domain.ErrNotFound):
		return "", err
	}

	if len(password) < minPasswordLength {
		return "", fmt.Errorf("auth: root admin password must be at least %d characters: %w", minPasswordLength, domain.ErrInvalid)
	}
	hash, err := uc.Hasher.Hash(password)
	if err != nil {
		return "", err
	}
	u = &domain.User{
		Email: email, PasswordHash: hash, StaffRole: domain.StaffSuperAdmin, Root: true, DisplayName: "Betailo", CreatedAt: uc.Clock.Now(),
	}
	if err := uc.Users.Create(ctx, u); err != nil {
		return "", err
	}
	return RootAdminCreated, uc.unmarkOthers(ctx, u)
}

// unmarkOthers keeps the root unique.
func (uc *EnsureRootAdmin) unmarkOthers(ctx context.Context, root *domain.User) error {
	users, err := uc.Users.List(ctx)
	if err != nil {
		return err
	}
	for _, other := range users {
		if other.Root && other.ID != root.ID {
			other.Root = false
			if err := uc.Users.Update(ctx, other); err != nil {
				return err
			}
		}
	}
	return nil
}

// HasSuperAdmin reports whether anyone can run the back office: false
// on a deployment that never configured its root account.
func (uc *EnsureRootAdmin) HasSuperAdmin(ctx context.Context) (bool, error) {
	users, err := uc.Users.List(ctx)
	if err != nil {
		return false, err
	}
	for _, u := range users {
		if u.StaffRole == domain.StaffSuperAdmin && !u.Suspended {
			return true, nil
		}
	}
	return false, nil
}

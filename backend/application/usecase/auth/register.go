// Package auth implements the registration and login usecases — a real
// auth module with working registration, login and JWT-verified
// middleware, completing Phase 1 of the roadmap (see "Betailo — Plan de
// développement"). Like application/usecase/reservation, both usecases
// depend only on application/ports interfaces, never on bcrypt, a JWT
// library, or mongo-driver directly.
package auth

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// selfRegisterableRoles is deliberately not domain.RoleAdmin — nobody
// becomes an admin by hitting a public registration endpoint.
var selfRegisterableRoles = map[domain.Role]bool{
	domain.RoleFarmer: true,
	domain.RoleBuyer:  true,
}

// Register is the usecase: validate input, ensure the email isn't
// already taken, hash the password, and persist the new user.
type Register struct {
	Users  ports.UserRepository
	Hasher ports.PasswordHasher
	Clock  ports.Clock
}

// RegisterInput is what a caller (the HTTP handler) supplies.
type RegisterInput struct {
	Email    string
	Password string
	Role     domain.Role
	// DisplayName is optional at sign-up (and editable later).
	DisplayName string
}

// RegisterOutput wraps the usecase's result.
type RegisterOutput struct {
	User *domain.User
}

// minPasswordLength is deliberately simple for the MVP — see the
// roadmap's post-MVP anti-fraud phase for anything stronger than a
// length floor.
const minPasswordLength = 8

// Execute validates in, then creates the user. It returns
// domain.ErrInvalid for a malformed input, domain.ErrEmailTaken if the
// email is already registered (surfaced by Users.Create, not checked
// separately here — see the comment above the call for why), and
// whatever error the repository or hasher returns otherwise.
func (uc *Register) Execute(ctx context.Context, in RegisterInput) (*RegisterOutput, error) {
	email := normalizeEmail(in.Email)
	if email == "" || !strings.Contains(email, "@") {
		return nil, domain.Coded("auth.email_invalid", fmt.Errorf("auth: email is required and must look like an email: %w", domain.ErrInvalid))
	}
	if len(in.Password) < minPasswordLength {
		return nil, domain.CodedWith("auth.password_too_short", map[string]any{"min": minPasswordLength}, fmt.Errorf("auth: password must be at least %d characters: %w", minPasswordLength, domain.ErrInvalid))
	}
	if !selfRegisterableRoles[in.Role] {
		return nil, fmt.Errorf("auth: role must be %q or %q: %w", domain.RoleFarmer, domain.RoleBuyer, domain.ErrInvalid)
	}

	hash, err := uc.Hasher.Hash(in.Password)
	if err != nil {
		return nil, err
	}

	displayName := strings.TrimSpace(in.DisplayName)
	if utf8.RuneCountInString(displayName) > domain.MaxDisplayNameLen {
		return nil, domain.CodedWith("account.display_name_too_long", map[string]any{"max": domain.MaxDisplayNameLen}, fmt.Errorf("auth: display name must be at most %d characters: %w", domain.MaxDisplayNameLen, domain.ErrInvalid))
	}

	user := &domain.User{
		Email:        email,
		PasswordHash: hash,
		Role:         in.Role,
		DisplayName:  displayName,
		CreatedAt:    uc.Clock.Now(),
	}

	// Create, not a separate FindByEmail-then-Create: doing the
	// existence check as its own step would race two concurrent
	// registrations for the same email (both check "not found", both
	// insert). Relying on the repository to turn the unique index
	// violation into domain.ErrEmailTaken (see
	// infrastructure/mongodb.UserRepository.Create) makes "email
	// already taken" atomic and correct under concurrency, the same
	// reasoning as the atomic decrement in
	// application/usecase/reservation.
	if err := uc.Users.Create(ctx, user); err != nil {
		return nil, err
	}

	return &RegisterOutput{User: user}, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

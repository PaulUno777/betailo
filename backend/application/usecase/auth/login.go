package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// Login is the usecase: verify credentials and issue an access token.
type Login struct {
	Users  ports.UserRepository
	Hasher ports.PasswordHasher
	Signer ports.TokenSigner
	Clock  ports.Clock
	TTL    time.Duration // access token lifetime; defaults to 24h if zero
}

// LoginInput is what a caller (the HTTP handler) supplies.
type LoginInput struct {
	Email    string
	Password string
}

// LoginOutput wraps the usecase's result.
type LoginOutput struct {
	AccessToken string
	User        *domain.User
}

// Execute verifies the email/password pair and, on success, signs an
// access token carrying the user's ID, email and role. It deliberately
// returns the same domain.ErrUnauthorized whether the email doesn't
// exist or the password is wrong — never let a login endpoint reveal
// which part of the pair was incorrect (that's a user-enumeration
// vector).
func (uc *Login) Execute(ctx context.Context, in LoginInput) (*LoginOutput, error) {
	email := normalizeEmail(in.Email)

	user, err := uc.Users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("auth: invalid credentials: %w", domain.ErrUnauthorized)
		}
		return nil, err
	}

	if err := uc.Hasher.Compare(user.PasswordHash, in.Password); err != nil {
		return nil, fmt.Errorf("auth: invalid credentials: %w", domain.ErrUnauthorized)
	}

	ttl := uc.TTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	now := uc.Clock.Now()

	token, err := uc.Signer.SignAccessToken(ctx, ports.TokenClaims{
		UserID:    user.ID,
		Email:     user.Email,
		Role:      user.Role,
		ExpiresAt: now.Add(ttl),
	})
	if err != nil {
		return nil, err
	}

	return &LoginOutput{AccessToken: token, User: user}, nil
}

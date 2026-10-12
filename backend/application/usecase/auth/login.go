package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// defaultRefreshTTL is 30 days — long enough that a "remembered" buyer
// or farmer isn't asked to re-login constantly, short enough that a
// stale, never-used cookie eventually falls out of Mongo's TTL index
// (see infrastructure/mongodb.ensureIndexes) on its own.
const defaultRefreshTTL = 30 * 24 * time.Hour

// Login is the usecase: verify credentials, issue an access token, and
// start a new refresh-token family (see docs/adr/0008 for why a
// refresh token exists at all — ADR-0006 originally deferred this).
type Login struct {
	Users         ports.UserRepository
	RefreshTokens ports.RefreshTokenRepository
	Hasher        ports.PasswordHasher
	Signer        ports.TokenSigner
	Clock         ports.Clock
	AccessTTL     time.Duration // access token lifetime; defaults to 24h if zero
	RefreshTTL    time.Duration // refresh token lifetime; defaults to defaultRefreshTTL if zero
}

// LoginInput is what a caller (the HTTP handler) supplies.
type LoginInput struct {
	Email    string
	Password string
	// Remember carried onto the new refresh-token record as
	// Persistent — the HTTP handler reads it back to decide whether
	// the cookie gets a real Expires or is session-only. The usecase
	// doesn't touch cookies itself (see ports.RefreshTokenRepository's
	// comment on why delivery stays in the handler).
	Remember bool
}

// LoginOutput wraps the usecase's result.
type LoginOutput struct {
	AccessToken      string
	RefreshToken     string // the raw secret; the handler puts it in the cookie, never logs it
	RefreshExpiresAt time.Time
	User             *domain.User
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
	// Refresh already refuses suspended accounts; without this a
	// suspended user simply logged in again.
	if user.Suspended {
		return nil, fmt.Errorf("auth: %w", domain.ErrSuspended)
	}

	accessTTL := uc.AccessTTL
	if accessTTL <= 0 {
		accessTTL = 24 * time.Hour
	}
	refreshTTL := uc.RefreshTTL
	if refreshTTL <= 0 {
		refreshTTL = defaultRefreshTTL
	}
	now := uc.Clock.Now()

	token, err := uc.Signer.SignAccessToken(ctx, ports.TokenClaims{
		UserID:    user.ID,
		Email:     user.Email,
		Role:      user.Role,
		ExpiresAt: now.Add(accessTTL),
	})
	if err != nil {
		return nil, err
	}

	rawRefresh, err := newRefreshSecret()
	if err != nil {
		return nil, err
	}
	refreshExpiresAt := now.Add(refreshTTL)
	if err := uc.RefreshTokens.Create(ctx, &ports.RefreshToken{
		ID:         bson.NewObjectID(),
		UserID:     user.ID,
		FamilyID:   bson.NewObjectID(),
		TokenHash:  hashRefreshToken(rawRefresh),
		IssuedAt:   now,
		ExpiresAt:  refreshExpiresAt,
		Persistent: in.Remember,
	}); err != nil {
		return nil, err
	}

	return &LoginOutput{
		AccessToken:      token,
		RefreshToken:     rawRefresh,
		RefreshExpiresAt: refreshExpiresAt,
		User:             user,
	}, nil
}

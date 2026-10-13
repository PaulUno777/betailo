package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	// AuthTokens holds the pending step of a two-factor sign-in.
	AuthTokens ports.AuthTokenRepository
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
	// Persistent is the session's "remember me", for the cookie.
	Persistent bool

	// MFARequired: the password was right and the account has a second
	// factor. No session was issued; MFAToken names the pending sign-in
	// for the second step and MFAMethods says which factors it takes.
	MFARequired bool
	MFAToken    string
	MFAMethods  []string
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

	if user.MFA.Enabled() && uc.AuthTokens != nil {
		raw, err := newRefreshSecret()
		if err != nil {
			return nil, err
		}
		now := uc.Clock.Now()
		if err := uc.AuthTokens.Create(ctx, &domain.AuthToken{
			Kind: domain.AuthTokenMFAChallenge, TokenHash: hashRefreshToken(raw), UserID: user.ID,
			Persistent: in.Remember, CreatedAt: now, ExpiresAt: now.Add(domain.MFAChallengeTTL),
		}); err != nil {
			return nil, err
		}
		return &LoginOutput{MFARequired: true, MFAToken: raw, MFAMethods: mfaMethods(user)}, nil
	}

	return uc.sessions().issue(ctx, user, in.Remember, false)
}

// SessionFor opens a fresh session for an account that just proved who
// it is another way (it changed its password with the current one).
// mfa carries over whether the session it replaces had passed a second
// factor.
func (uc *Login) SessionFor(ctx context.Context, user *domain.User, remember, mfa bool) (*LoginOutput, error) {
	return uc.sessions().issue(ctx, user, remember, mfa)
}

func (uc *Login) sessions() sessionIssuer {
	return sessionIssuer{RefreshTokens: uc.RefreshTokens, Signer: uc.Signer, Clock: uc.Clock, AccessTTL: uc.AccessTTL, RefreshTTL: uc.RefreshTTL}
}

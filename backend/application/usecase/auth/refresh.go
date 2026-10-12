package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// RefreshToken is the usecase behind POST /auth/refresh: rotate a
// still-valid refresh token for a new access token (and a new refresh
// token in the same family), or — if the one presented was already
// rotated — revoke the whole family as a token-theft signal. See
// docs/adr/0008 for why this exists at all.
type RefreshToken struct {
	Users         ports.UserRepository
	RefreshTokens ports.RefreshTokenRepository
	Signer        ports.TokenSigner
	Clock         ports.Clock
	AccessTTL     time.Duration // defaults to 24h if zero, matching Login
	RefreshTTL    time.Duration // defaults to defaultRefreshTTL if zero, matching Login
}

// RefreshTokenInput is what the HTTP handler supplies — the raw secret
// read from the httpOnly cookie, never anything else about the
// request (no user ID: that would let a caller ask to refresh on
// someone else's behalf just by guessing an ID).
type RefreshTokenInput struct {
	RawToken string
}

// RefreshTokenOutput wraps the usecase's result. User is included so
// the handler can refresh its own view of the account without a
// second round trip, but is deliberately not echoed back to the
// client in the response body (see auth_handlers.go) — a refresh call
// only needs to return a new access token.
type RefreshTokenOutput struct {
	AccessToken      string
	RefreshToken     string
	RefreshExpiresAt time.Time
	Persistent       bool
	User             *domain.User
}

// Execute rejects with domain.ErrUnauthorized for every failure mode a
// client can observe (unknown, expired, reused, or the owning account
// gone/suspended) — deliberately indistinguishable from the outside,
// the same anti-enumeration reasoning Login.Execute already applies to
// credentials. Reuse detection (a token presented again after it was
// already rotated) is logged distinctly server-side since it's a
// stronger signal than an ordinary expired cookie.
func (uc *RefreshToken) Execute(ctx context.Context, in RefreshTokenInput) (*RefreshTokenOutput, error) {
	if in.RawToken == "" {
		return nil, fmt.Errorf("auth: refresh token required: %w", domain.ErrUnauthorized)
	}

	stored, err := uc.RefreshTokens.FindByHash(ctx, hashRefreshToken(in.RawToken))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("auth: unknown refresh token: %w", domain.ErrUnauthorized)
		}
		return nil, err
	}

	now := uc.Clock.Now()

	if stored.RevokedAt != nil {
		// Already rotated (or already logged out) and presented again:
		// the only way that happens honestly is a race between two
		// requests rotating the same token at once, which a client
		// should simply retry as a fresh login — so treat it the same
		// as theft rather than trying to special-case the benign race.
		// Kill the whole chain, not just this token.
		if revokeErr := uc.RefreshTokens.RevokeFamily(ctx, stored.FamilyID, "reuse_detected", now); revokeErr != nil {
			return nil, revokeErr
		}
		slog.Warn("auth: refresh token reuse detected, family revoked",
			slog.String("familyId", stored.FamilyID.Hex()), slog.String("userId", stored.UserID.Hex()))
		return nil, fmt.Errorf("auth: refresh token already used: %w", domain.ErrUnauthorized)
	}

	if now.After(stored.ExpiresAt) {
		return nil, fmt.Errorf("auth: refresh token expired: %w", domain.ErrUnauthorized)
	}

	// Reload the user fresh rather than trusting anything cached — a
	// role change or suspension since the last access token was signed
	// takes effect right here, which is the actual point of this
	// usecase existing (see ADR-0006's named consequence and ADR-0008).
	user, err := uc.Users.FindByID(ctx, stored.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("auth: account no longer exists: %w", domain.ErrUnauthorized)
		}
		return nil, err
	}
	if user.Suspended {
		return nil, fmt.Errorf("auth: account suspended: %w", domain.ErrUnauthorized)
	}

	accessTTL := uc.AccessTTL
	if accessTTL <= 0 {
		accessTTL = 24 * time.Hour
	}
	refreshTTL := uc.RefreshTTL
	if refreshTTL <= 0 {
		refreshTTL = defaultRefreshTTL
	}

	accessToken, err := uc.Signer.SignAccessToken(ctx, ports.TokenClaims{
		UserID:    user.ID,
		Email:     user.Email,
		StaffRole: user.StaffRole,
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
	next := &ports.RefreshToken{
		ID:         bson.NewObjectID(),
		UserID:     stored.UserID,
		FamilyID:   stored.FamilyID,
		TokenHash:  hashRefreshToken(rawRefresh),
		IssuedAt:   now,
		ExpiresAt:  refreshExpiresAt,
		Persistent: stored.Persistent, // carried through the chain — see ports.RefreshToken's comment
	}
	if err := uc.RefreshTokens.Rotate(ctx, stored.ID, next); err != nil {
		return nil, err
	}

	return &RefreshTokenOutput{
		AccessToken:      accessToken,
		RefreshToken:     rawRefresh,
		RefreshExpiresAt: refreshExpiresAt,
		Persistent:       stored.Persistent,
		User:             user,
	}, nil
}

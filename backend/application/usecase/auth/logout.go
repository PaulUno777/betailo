package auth

import (
	"context"
	"errors"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// Logout is the usecase behind POST /auth/logout: revoke the current
// device's refresh-token family. Deliberately scoped to one device —
// "sign out everywhere" would need ports.RefreshTokenRepository to
// grow a RevokeAllForUser method, which doesn't exist yet because
// nothing calls it yet (see that port's comment).
type Logout struct {
	RefreshTokens ports.RefreshTokenRepository
	Clock         ports.Clock
}

// LogoutInput is what the HTTP handler supplies — the raw secret read
// from the httpOnly cookie, or empty if there was none to begin with.
type LogoutInput struct {
	RawToken string
}

// Execute is idempotent by design: logging out twice, or logging out
// with an empty, unknown, or already-revoked token, is always a
// successful no-op from the client's point of view — the handler
// clears the cookie either way (see auth_handlers.go), so there is
// nothing for an error here to usefully report.
func (uc *Logout) Execute(ctx context.Context, in LogoutInput) error {
	if in.RawToken == "" {
		return nil
	}

	stored, err := uc.RefreshTokens.FindByHash(ctx, hashRefreshToken(in.RawToken))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		return err
	}

	return uc.RefreshTokens.RevokeFamily(ctx, stored.FamilyID, "logout", uc.Clock.Now())
}

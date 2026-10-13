package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// PasswordReset lets someone who lost their password choose a new one
// through a link sent to the account's address. It is also the root
// account's way back in.
type PasswordReset struct {
	Users         ports.UserRepository
	AuthTokens    ports.AuthTokenRepository
	RefreshTokens ports.RefreshTokenRepository
	Hasher        ports.PasswordHasher
	Mailer        ports.Mailer
	Clock         ports.Clock
	PublicURL     string
	Logger        *slog.Logger
}

// Request e-mails a reset link when the address has an account. It
// returns nil either way: the answer must not say whether an address is
// registered.
func (uc *PasswordReset) Request(ctx context.Context, email string) error {
	u, err := uc.Users.FindByEmail(ctx, normalizeEmail(email))
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if u.Suspended {
		return nil
	}
	raw, err := newRefreshSecret()
	if err != nil {
		return err
	}
	now := uc.Clock.Now()
	if err := uc.AuthTokens.Create(ctx, &domain.AuthToken{
		Kind: domain.AuthTokenPasswordReset, TokenHash: hashRefreshToken(raw), UserID: u.ID,
		CreatedAt: now, ExpiresAt: now.Add(domain.PasswordResetTTL),
	}); err != nil {
		return err
	}
	link := strings.TrimRight(uc.PublicURL, "/") + "/reset-password/" + raw
	if err := uc.Mailer.Send(ctx, ports.Mail{
		To:      u.Email,
		Subject: "Réinitialisation de votre mot de passe Betailo",
		Text: fmt.Sprintf("Bonjour,\n\nUne réinitialisation du mot de passe a été demandée pour votre compte Betailo.\n\n"+
			"Ouvrez ce lien pour choisir un nouveau mot de passe :\n%s\n\n"+
			"Le lien est valable 1 heure et ne sert qu’une fois. Si vous n’êtes pas à l’origine de cette demande, ignorez ce message : votre mot de passe ne change pas.\n", link),
	}); err != nil && uc.Logger != nil {
		uc.Logger.Error("auth: password reset mail not sent", slog.Any("err", err))
	}
	return nil
}

// Reset sets a new password from a reset link and ends every session
// of the account: whoever knew the old password is out.
func (uc *PasswordReset) Reset(ctx context.Context, rawToken, newPassword string) error {
	gone := domain.Coded("auth.reset_link_invalid", fmt.Errorf("auth: this reset link is no longer valid: %w", domain.ErrNotFound))
	if rawToken == "" {
		return gone
	}
	t, err := uc.AuthTokens.FindByHash(ctx, domain.AuthTokenPasswordReset, hashRefreshToken(rawToken))
	now := uc.Clock.Now()
	if err != nil || !t.Open(now) {
		return gone
	}
	if len(newPassword) < minPasswordLength {
		return domain.CodedWith("auth.password_too_short", map[string]any{"min": minPasswordLength}, fmt.Errorf("auth: password must be at least %d characters: %w", minPasswordLength, domain.ErrInvalid))
	}
	u, err := uc.Users.FindByID(ctx, t.UserID)
	if err != nil {
		return gone
	}
	hash, err := uc.Hasher.Hash(newPassword)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	if err := uc.Users.Update(ctx, u); err != nil {
		return err
	}
	t.UsedAt = &now
	if err := uc.AuthTokens.Update(ctx, t); err != nil {
		return err
	}
	return uc.RefreshTokens.RevokeAllForUser(ctx, u.ID, "password_reset", now)
}

package auth

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// sessionIssuer opens a session for an account whose identity is
// established: an access token and the first refresh token of a new
// family. Login uses it directly; the second step of a two-factor
// sign-in and the acceptance of an invitation end with it too.
type sessionIssuer struct {
	RefreshTokens ports.RefreshTokenRepository
	Signer        ports.TokenSigner
	Clock         ports.Clock
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

func (s sessionIssuer) ttls() (time.Duration, time.Duration) {
	access, refresh := s.AccessTTL, s.RefreshTTL
	if access <= 0 {
		access = 24 * time.Hour
	}
	if refresh <= 0 {
		refresh = defaultRefreshTTL
	}
	return access, refresh
}

// issue opens the session. mfa records that a second factor was passed.
func (s sessionIssuer) issue(ctx context.Context, user *domain.User, persistent, mfa bool) (*LoginOutput, error) {
	accessTTL, refreshTTL := s.ttls()
	now := s.Clock.Now()

	token, err := s.Signer.SignAccessToken(ctx, ports.TokenClaims{
		UserID: user.ID, Email: user.Email, StaffRole: user.StaffRole, MFA: mfa, ExpiresAt: now.Add(accessTTL),
	})
	if err != nil {
		return nil, err
	}
	rawRefresh, err := newRefreshSecret()
	if err != nil {
		return nil, err
	}
	refreshExpiresAt := now.Add(refreshTTL)
	if err := s.RefreshTokens.Create(ctx, &ports.RefreshToken{
		ID: bson.NewObjectID(), UserID: user.ID, FamilyID: bson.NewObjectID(), TokenHash: hashRefreshToken(rawRefresh),
		IssuedAt: now, ExpiresAt: refreshExpiresAt, Persistent: persistent, MFA: mfa,
	}); err != nil {
		return nil, err
	}
	return &LoginOutput{AccessToken: token, RefreshToken: rawRefresh, RefreshExpiresAt: refreshExpiresAt, User: user, Persistent: persistent}, nil
}

// upgrade marks the session behind rawRefresh as having passed a second
// factor and returns an access token saying so: the account enrolled a
// factor while signed in. Without a live refresh token only the access
// token carries it, until it expires.
func (s sessionIssuer) upgrade(ctx context.Context, user *domain.User, rawRefresh string) (string, error) {
	accessTTL, _ := s.ttls()
	now := s.Clock.Now()
	if rawRefresh != "" {
		if stored, err := s.RefreshTokens.FindByHash(ctx, hashRefreshToken(rawRefresh)); err == nil &&
			stored.UserID == user.ID && stored.RevokedAt == nil && now.Before(stored.ExpiresAt) {
			if err := s.RefreshTokens.SetMFA(ctx, stored.ID); err != nil {
				return "", err
			}
		}
	}
	return s.Signer.SignAccessToken(ctx, ports.TokenClaims{
		UserID: user.ID, Email: user.Email, StaffRole: user.StaffRole, MFA: true, ExpiresAt: now.Add(accessTTL),
	})
}

// MFA method names, as clients read them.
const (
	MFAMethodTOTP     = "totp"
	MFAMethodPasskey  = "passkey"
	MFAMethodRecovery = "recovery"
)

func mfaMethods(u *domain.User) []string {
	var methods []string
	if u.MFA.HasTOTP() {
		methods = append(methods, MFAMethodTOTP)
	}
	if len(u.MFA.Passkeys) > 0 {
		methods = append(methods, MFAMethodPasskey)
	}
	if len(u.MFA.RecoveryHashes) > 0 {
		methods = append(methods, MFAMethodRecovery)
	}
	return methods
}

package ports

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

// TokenClaims is what a verified access token carries — trusted
// directly by the auth middleware and by usecases behind it, with no
// database round trip needed to re-derive whether they are staff.
type TokenClaims struct {
	UserID    bson.ObjectID
	Email     string
	StaffRole domain.StaffRole
	// MFA is true when the session behind the token passed a second
	// factor (or enrolled one). Staff permissions ask for it.
	MFA       bool
	ExpiresAt time.Time
}

// TokenSigner abstracts access-token issuance and verification so
// usecases and the httpserver middleware never depend on a specific JWT
// library directly.
type TokenSigner interface {
	// SignAccessToken issues a token encoding claims, valid until
	// claims.ExpiresAt.
	SignAccessToken(ctx context.Context, claims TokenClaims) (token string, err error)

	// ParseAccessToken verifies a token's signature and expiry and
	// returns its claims. Implementations return domain.ErrUnauthorized
	// for any invalid, expired, or tampered token — never a
	// library-specific error a caller would need to unwrap.
	ParseAccessToken(ctx context.Context, token string) (TokenClaims, error)
}

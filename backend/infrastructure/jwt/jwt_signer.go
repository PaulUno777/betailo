// Package jwt implements application/ports.TokenSigner via
// github.com/golang-jwt/jwt/v5, HMAC-SHA256. It is the only package in
// this codebase that imports a JWT library — application/usecase/auth
// and the httpserver middleware depend only on ports.TokenSigner.
package jwt

import (
	"context"
	"fmt"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// claims is the on-the-wire JWT payload shape. It stays private to this
// file — ports.TokenClaims is the type the rest of the codebase works
// with; mapping between the two happens only here.
type claims struct {
	StaffRole domain.StaffRole `json:"staffRole,omitempty"`
	Email     string           `json:"email"`
	jwtlib.RegisteredClaims
}

// Signer implements ports.TokenSigner with a shared HMAC secret.
type Signer struct {
	// Secret signs and verifies every token. Losing it invalidates
	// every issued token; rotating it does too — there is no key-ID
	// support here, deliberately, for an MVP with one signing key.
	Secret []byte
}

var _ ports.TokenSigner = (*Signer)(nil)

// SignAccessToken implements ports.TokenSigner.
func (s *Signer) SignAccessToken(_ context.Context, c ports.TokenClaims) (string, error) {
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims{
		StaffRole: c.StaffRole,
		Email:     c.Email,
		RegisteredClaims: jwtlib.RegisteredClaims{
			Subject:   c.UserID.Hex(),
			ExpiresAt: jwtlib.NewNumericDate(c.ExpiresAt),
			IssuedAt:  jwtlib.NewNumericDate(time.Now().UTC()),
		},
	})
	return token.SignedString(s.Secret)
}

// ParseAccessToken implements ports.TokenSigner. Every failure mode —
// bad signature, wrong algorithm, expired, malformed subject — maps to
// domain.ErrUnauthorized; callers never need to know which.
func (s *Signer) ParseAccessToken(_ context.Context, token string) (ports.TokenClaims, error) {
	var c claims
	parsed, err := jwtlib.ParseWithClaims(token, &c, func(t *jwtlib.Token) (any, error) {
		if _, ok := t.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("jwt: unexpected signing method %v", t.Header["alg"])
		}
		return s.Secret, nil
	})
	if err != nil || !parsed.Valid {
		return ports.TokenClaims{}, domain.ErrUnauthorized
	}

	userID, err := bson.ObjectIDFromHex(c.Subject)
	if err != nil {
		return ports.TokenClaims{}, domain.ErrUnauthorized
	}
	if c.ExpiresAt == nil {
		return ports.TokenClaims{}, domain.ErrUnauthorized
	}

	return ports.TokenClaims{
		UserID:    userID,
		Email:     c.Email,
		StaffRole: c.StaffRole,
		ExpiresAt: c.ExpiresAt.Time,
	}, nil
}

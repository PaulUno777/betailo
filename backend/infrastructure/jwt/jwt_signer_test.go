package jwt

import (
	"context"
	"errors"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

func TestSigner_RoundTrip(t *testing.T) {
	s := &Signer{Secret: []byte("test-secret")}
	ctx := context.Background()
	in := ports.TokenClaims{
		UserID:    bson.NewObjectID(),
		Email:     "staff@example.com",
		StaffRole: domain.StaffSuperAdmin,
		ExpiresAt: time.Now().Add(time.Hour).UTC().Truncate(time.Second),
	}

	token, err := s.SignAccessToken(ctx, in)
	if err != nil {
		t.Fatalf("SignAccessToken() error = %v", err)
	}
	if token == "" {
		t.Fatal("SignAccessToken() returned an empty token")
	}

	out, err := s.ParseAccessToken(ctx, token)
	if err != nil {
		t.Fatalf("ParseAccessToken() error = %v", err)
	}
	if out.UserID != in.UserID {
		t.Errorf("UserID = %v, want %v", out.UserID, in.UserID)
	}
	if out.Email != in.Email {
		t.Errorf("Email = %q, want %q", out.Email, in.Email)
	}
	if out.StaffRole != in.StaffRole {
		t.Errorf("StaffRole = %v, want %v", out.StaffRole, in.StaffRole)
	}
	if !out.ExpiresAt.Equal(in.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", out.ExpiresAt, in.ExpiresAt)
	}
}

func TestSigner_ParseAccessToken_Expired(t *testing.T) {
	s := &Signer{Secret: []byte("test-secret")}
	ctx := context.Background()

	token, err := s.SignAccessToken(ctx, ports.TokenClaims{
		UserID:    bson.NewObjectID(),
		ExpiresAt: time.Now().Add(-time.Minute), // already expired
	})
	if err != nil {
		t.Fatalf("SignAccessToken() error = %v", err)
	}

	_, err = s.ParseAccessToken(ctx, token)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("ParseAccessToken() error = %v, want domain.ErrUnauthorized", err)
	}
}

func TestSigner_ParseAccessToken_WrongSecret(t *testing.T) {
	ctx := context.Background()
	signed := &Signer{Secret: []byte("secret-a")}
	token, err := signed.SignAccessToken(ctx, ports.TokenClaims{
		UserID:    bson.NewObjectID(),
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("SignAccessToken() error = %v", err)
	}

	verifier := &Signer{Secret: []byte("secret-b")}
	_, err = verifier.ParseAccessToken(ctx, token)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("ParseAccessToken() with wrong secret: error = %v, want domain.ErrUnauthorized", err)
	}
}

func TestSigner_ParseAccessToken_Malformed(t *testing.T) {
	s := &Signer{Secret: []byte("test-secret")}
	_, err := s.ParseAccessToken(context.Background(), "not-a-jwt")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("ParseAccessToken() error = %v, want domain.ErrUnauthorized", err)
	}
}

func TestSigner_ParseAccessToken_RejectsAlgNone(t *testing.T) {
	// A token signed with "none" is a classic JWT library footgun: if
	// ParseAccessToken didn't pin the expected algorithm family, an
	// attacker could forge a token nobody actually signed.
	s := &Signer{Secret: []byte("test-secret")}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodNone, claims{
		StaffRole: domain.StaffSuperAdmin,
		Email:     "attacker@example.com",
		RegisteredClaims: jwtlib.RegisteredClaims{
			Subject:   bson.NewObjectID().Hex(),
			ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	unsigned, err := token.SignedString(jwtlib.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	_, err = s.ParseAccessToken(context.Background(), unsigned)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("ParseAccessToken() of an alg=none token: error = %v, want domain.ErrUnauthorized", err)
	}
}

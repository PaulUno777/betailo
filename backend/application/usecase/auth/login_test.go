package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pauluno777/betailo/backend/domain"
)

func registeredUser(t *testing.T, users *fakeUsers, email, password string, role domain.Role) *domain.User {
	t.Helper()
	hash, err := (fakeHasher{}).Hash(password)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	u := &domain.User{Email: normalizeEmail(email), PasswordHash: hash, Role: role, CreatedAt: time.Now()}
	if err := users.Create(context.Background(), u); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	return u
}

func TestLogin_Execute_Success(t *testing.T) {
	users := newFakeUsers()
	registeredUser(t, users, "farmer@example.com", "correct-horse", domain.RoleFarmer)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	signer := &fakeSigner{}
	refreshTokens := newFakeRefreshTokens()
	uc := &Login{
		Users: users, RefreshTokens: refreshTokens, Hasher: fakeHasher{}, Signer: signer,
		Clock: fakeClock{now: now}, AccessTTL: time.Hour, RefreshTTL: 48 * time.Hour,
	}

	out, err := uc.Execute(context.Background(), LoginInput{Email: " Farmer@Example.com ", Password: "correct-horse", Remember: true})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if out.AccessToken != "signed-token" {
		t.Errorf("AccessToken = %q, want %q", out.AccessToken, "signed-token")
	}
	if out.User.Email != "farmer@example.com" {
		t.Errorf("User.Email = %q, want %q", out.User.Email, "farmer@example.com")
	}
	if len(signer.signed) != 1 {
		t.Fatalf("signed %d tokens, want 1", len(signer.signed))
	}
	claims := signer.signed[0]
	if claims.Email != "farmer@example.com" || claims.Role != domain.RoleFarmer {
		t.Errorf("claims = %+v, want email/role to match the user", claims)
	}
	if want := now.Add(time.Hour); !claims.ExpiresAt.Equal(want) {
		t.Errorf("claims.ExpiresAt = %v, want %v", claims.ExpiresAt, want)
	}

	// The refresh token: a non-empty raw secret, matching ExpiresAt,
	// and a record actually persisted with Persistent carrying
	// Remember through.
	if out.RefreshToken == "" {
		t.Fatal("RefreshToken is empty, want a raw secret")
	}
	if want := now.Add(48 * time.Hour); !out.RefreshExpiresAt.Equal(want) {
		t.Errorf("RefreshExpiresAt = %v, want %v", out.RefreshExpiresAt, want)
	}
	stored, err := refreshTokens.FindByHash(context.Background(), hashRefreshToken(out.RefreshToken))
	if err != nil {
		t.Fatalf("stored refresh token not found: %v", err)
	}
	if stored.UserID != out.User.ID {
		t.Errorf("stored.UserID = %v, want %v", stored.UserID, out.User.ID)
	}
	if !stored.Persistent {
		t.Error("stored.Persistent = false, want true (Remember: true)")
	}
}

func TestLogin_Execute_DefaultTTL(t *testing.T) {
	users := newFakeUsers()
	registeredUser(t, users, "a@example.com", "correct-horse", domain.RoleBuyer)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	signer := &fakeSigner{}
	uc := &Login{Users: users, RefreshTokens: newFakeRefreshTokens(), Hasher: fakeHasher{}, Signer: signer, Clock: fakeClock{now: now}} // AccessTTL/RefreshTTL left zero

	out, err := uc.Execute(context.Background(), LoginInput{Email: "a@example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if want := now.Add(24 * time.Hour); !signer.signed[0].ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want default 24h from now (%v)", signer.signed[0].ExpiresAt, want)
	}
	if want := now.Add(defaultRefreshTTL); !out.RefreshExpiresAt.Equal(want) {
		t.Errorf("RefreshExpiresAt = %v, want default 30d from now (%v)", out.RefreshExpiresAt, want)
	}
}

func TestLogin_Execute_WrongPassword(t *testing.T) {
	users := newFakeUsers()
	registeredUser(t, users, "a@example.com", "correct-horse", domain.RoleBuyer)
	uc := &Login{Users: users, RefreshTokens: newFakeRefreshTokens(), Hasher: fakeHasher{}, Signer: &fakeSigner{}, Clock: fakeClock{now: time.Now()}}

	_, err := uc.Execute(context.Background(), LoginInput{Email: "a@example.com", Password: "wrong-password"})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("Execute() error = %v, want domain.ErrUnauthorized", err)
	}
}

func TestLogin_Execute_UnknownEmail(t *testing.T) {
	uc := &Login{Users: newFakeUsers(), RefreshTokens: newFakeRefreshTokens(), Hasher: fakeHasher{}, Signer: &fakeSigner{}, Clock: fakeClock{now: time.Now()}}

	_, err := uc.Execute(context.Background(), LoginInput{Email: "nobody@example.com", Password: "correct-horse"})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("Execute() error = %v, want domain.ErrUnauthorized (never leak that the account doesn't exist)", err)
	}
}

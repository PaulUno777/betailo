package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pauluno777/betailo/backend/domain"
)

// loginFor is a small helper: register+login a user and return both
// the user and the raw refresh secret login issued, so each test below
// starts from a real token family rather than hand-building one.
func loginFor(t *testing.T, users *fakeUsers, refreshTokens *fakeRefreshTokens, signer *fakeSigner, now time.Time, email string, role domain.Role, remember bool) (*domain.User, string) {
	t.Helper()
	registeredUser(t, users, email, "correct-horse", role)
	loginUC := &Login{
		Users: users, RefreshTokens: refreshTokens, Hasher: fakeHasher{}, Signer: signer,
		Clock: fakeClock{now: now}, AccessTTL: time.Hour, RefreshTTL: 48 * time.Hour,
	}
	out, err := loginUC.Execute(context.Background(), LoginInput{Email: email, Password: "correct-horse", Remember: remember})
	if err != nil {
		t.Fatalf("login Execute() error = %v", err)
	}
	return out.User, out.RefreshToken
}

func TestRefreshToken_Execute_Success(t *testing.T) {
	users := newFakeUsers()
	refreshTokens := newFakeRefreshTokens()
	signer := &fakeSigner{}
	loginAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	user, rawRefresh := loginFor(t, users, refreshTokens, signer, loginAt, "farmer@example.com", domain.RoleFarmer, true)

	// Simulate a role change between login and refresh — this is the
	// actual point of reloading the user fresh in RefreshToken.Execute
	// rather than trusting the claims a prior access token carried.
	user.Role = domain.RoleAdmin
	if err := users.Update(context.Background(), user); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	refreshAt := loginAt.Add(time.Minute)
	uc := &RefreshToken{
		Users: users, RefreshTokens: refreshTokens, Signer: signer,
		Clock: fakeClock{now: refreshAt}, AccessTTL: time.Hour, RefreshTTL: 48 * time.Hour,
	}
	out, err := uc.Execute(context.Background(), RefreshTokenInput{RawToken: rawRefresh})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	if out.AccessToken == "" || out.RefreshToken == "" {
		t.Fatal("expected both a new access token and a new refresh token")
	}
	if out.RefreshToken == rawRefresh {
		t.Error("RefreshToken should be a freshly rotated secret, not the one presented")
	}
	if !out.Persistent {
		t.Error("Persistent = false, want true (carried from the original login's Remember)")
	}

	// The role change took effect: the newly signed claims reflect it.
	last := signer.signed[len(signer.signed)-1]
	if last.Role != domain.RoleAdmin {
		t.Errorf("signed claims.Role = %v, want %v (role change should propagate on refresh)", last.Role, domain.RoleAdmin)
	}

	// The old secret is now dead — rotate-on-use.
	old, err := refreshTokens.FindByHash(context.Background(), hashRefreshToken(rawRefresh))
	if err != nil {
		t.Fatalf("old token record missing: %v", err)
	}
	if old.RevokedAt == nil || old.RevokedReason != "rotated" {
		t.Errorf("old token RevokedAt/Reason = %v/%q, want revoked with reason %q", old.RevokedAt, old.RevokedReason, "rotated")
	}
}

func TestRefreshToken_Execute_ReuseRevokesWholeFamily(t *testing.T) {
	users := newFakeUsers()
	refreshTokens := newFakeRefreshTokens()
	signer := &fakeSigner{}
	loginAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, rawRefresh1 := loginFor(t, users, refreshTokens, signer, loginAt, "buyer@example.com", domain.RoleBuyer, false)

	uc := &RefreshToken{Users: users, RefreshTokens: refreshTokens, Signer: signer, Clock: fakeClock{now: loginAt.Add(time.Minute)}}

	// First refresh: legitimate rotation, rawRefresh1 -> rawRefresh2.
	out1, err := uc.Execute(context.Background(), RefreshTokenInput{RawToken: rawRefresh1})
	if err != nil {
		t.Fatalf("first Execute() error = %v, want nil", err)
	}
	rawRefresh2 := out1.RefreshToken

	// Presenting rawRefresh1 again (already rotated, e.g. stolen and
	// replayed) must be rejected AND kill the whole family — including
	// rawRefresh2, the token that's actually still "valid" by itself.
	uc2 := &RefreshToken{Users: users, RefreshTokens: refreshTokens, Signer: signer, Clock: fakeClock{now: loginAt.Add(2 * time.Minute)}}
	if _, err := uc2.Execute(context.Background(), RefreshTokenInput{RawToken: rawRefresh1}); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("reuse Execute() error = %v, want domain.ErrUnauthorized", err)
	}

	// The key property: rawRefresh2 — never itself reused, never
	// expired — is now also dead, because reuse detection revokes the
	// entire family, not just the replayed token.
	uc3 := &RefreshToken{Users: users, RefreshTokens: refreshTokens, Signer: signer, Clock: fakeClock{now: loginAt.Add(3 * time.Minute)}}
	if _, err := uc3.Execute(context.Background(), RefreshTokenInput{RawToken: rawRefresh2}); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("post-reuse Execute() with the latest valid token error = %v, want domain.ErrUnauthorized (whole family should be dead)", err)
	}
}

func TestRefreshToken_Execute_Expired(t *testing.T) {
	users := newFakeUsers()
	refreshTokens := newFakeRefreshTokens()
	signer := &fakeSigner{}
	loginAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, rawRefresh := loginFor(t, users, refreshTokens, signer, loginAt, "buyer@example.com", domain.RoleBuyer, false)

	// Past the 30-day default RefreshTTL loginFor's Login used.
	uc := &RefreshToken{Users: users, RefreshTokens: refreshTokens, Signer: signer, Clock: fakeClock{now: loginAt.Add(31 * 24 * time.Hour)}}
	if _, err := uc.Execute(context.Background(), RefreshTokenInput{RawToken: rawRefresh}); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("Execute() error = %v, want domain.ErrUnauthorized", err)
	}
}

func TestRefreshToken_Execute_SuspendedUser(t *testing.T) {
	users := newFakeUsers()
	refreshTokens := newFakeRefreshTokens()
	signer := &fakeSigner{}
	loginAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	user, rawRefresh := loginFor(t, users, refreshTokens, signer, loginAt, "farmer@example.com", domain.RoleFarmer, false)

	user.Suspended = true
	if err := users.Update(context.Background(), user); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	uc := &RefreshToken{Users: users, RefreshTokens: refreshTokens, Signer: signer, Clock: fakeClock{now: loginAt.Add(time.Minute)}}
	if _, err := uc.Execute(context.Background(), RefreshTokenInput{RawToken: rawRefresh}); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("Execute() error = %v, want domain.ErrUnauthorized (a still-valid refresh token must not survive a suspension)", err)
	}
}

func TestRefreshToken_Execute_UnknownToken(t *testing.T) {
	uc := &RefreshToken{Users: newFakeUsers(), RefreshTokens: newFakeRefreshTokens(), Signer: &fakeSigner{}, Clock: fakeClock{now: time.Now()}}
	if _, err := uc.Execute(context.Background(), RefreshTokenInput{RawToken: "never-issued"}); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("Execute() error = %v, want domain.ErrUnauthorized", err)
	}
}

func TestRefreshToken_Execute_EmptyToken(t *testing.T) {
	uc := &RefreshToken{Users: newFakeUsers(), RefreshTokens: newFakeRefreshTokens(), Signer: &fakeSigner{}, Clock: fakeClock{now: time.Now()}}
	if _, err := uc.Execute(context.Background(), RefreshTokenInput{RawToken: ""}); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("Execute() error = %v, want domain.ErrUnauthorized", err)
	}
}

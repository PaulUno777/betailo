package auth

import (
	"context"
	"testing"
	"time"
)

func TestLogout_Execute_RevokesFamily(t *testing.T) {
	users := newFakeUsers()
	refreshTokens := newFakeRefreshTokens()
	signer := &fakeSigner{}
	loginAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, rawRefresh := loginFor(t, users, refreshTokens, signer, loginAt, "buyer@example.com", false)

	logoutUC := &Logout{RefreshTokens: refreshTokens, Clock: fakeClock{now: loginAt.Add(time.Minute)}}
	if err := logoutUC.Execute(context.Background(), LogoutInput{RawToken: rawRefresh}); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	stored, err := refreshTokens.FindByHash(context.Background(), hashRefreshToken(rawRefresh))
	if err != nil {
		t.Fatalf("FindByHash() error = %v", err)
	}
	if stored.RevokedAt == nil || stored.RevokedReason != "logout" {
		t.Errorf("RevokedAt/Reason = %v/%q, want revoked with reason %q", stored.RevokedAt, stored.RevokedReason, "logout")
	}

	// A revoked token can no longer refresh.
	refreshUC := &RefreshToken{Users: users, RefreshTokens: refreshTokens, Signer: signer, Clock: fakeClock{now: loginAt.Add(2 * time.Minute)}}
	if _, err := refreshUC.Execute(context.Background(), RefreshTokenInput{RawToken: rawRefresh}); err == nil {
		t.Error("Execute() after logout succeeded, want an error (revoked token must not refresh)")
	}
}

func TestLogout_Execute_IdempotentOnEmptyOrUnknownToken(t *testing.T) {
	logoutUC := &Logout{RefreshTokens: newFakeRefreshTokens(), Clock: fakeClock{now: time.Now()}}

	if err := logoutUC.Execute(context.Background(), LogoutInput{RawToken: ""}); err != nil {
		t.Errorf("Execute(empty) error = %v, want nil", err)
	}
	if err := logoutUC.Execute(context.Background(), LogoutInput{RawToken: "never-issued"}); err != nil {
		t.Errorf("Execute(unknown) error = %v, want nil", err)
	}
}

func TestLogout_Execute_IdempotentOnAlreadyRevoked(t *testing.T) {
	users := newFakeUsers()
	refreshTokens := newFakeRefreshTokens()
	signer := &fakeSigner{}
	loginAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, rawRefresh := loginFor(t, users, refreshTokens, signer, loginAt, "buyer@example.com", false)

	logoutUC := &Logout{RefreshTokens: refreshTokens, Clock: fakeClock{now: loginAt.Add(time.Minute)}}
	if err := logoutUC.Execute(context.Background(), LogoutInput{RawToken: rawRefresh}); err != nil {
		t.Fatalf("first Execute() error = %v, want nil", err)
	}
	if err := logoutUC.Execute(context.Background(), LogoutInput{RawToken: rawRefresh}); err != nil {
		t.Errorf("second Execute() error = %v, want nil (logout is idempotent)", err)
	}
}

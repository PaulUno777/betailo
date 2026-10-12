package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pauluno777/betailo/backend/domain"
)

func newRootAdmin(users *fakeUsers) *EnsureRootAdmin {
	return &EnsureRootAdmin{Users: users, Hasher: fakeHasher{}, Clock: fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}
}

func TestEnsureRootAdmin_CreatesTheAccountOnDayZero(t *testing.T) {
	users := newFakeUsers()
	uc := newRootAdmin(users)

	outcome, err := uc.Execute(context.Background(), " Root@Betailo.test ", "a-long-password")
	if err != nil || outcome != RootAdminCreated {
		t.Fatalf("Execute() = %q, %v; want created", outcome, err)
	}
	root := users.byEmail["root@betailo.test"]
	if root == nil || !root.Root || root.StaffRole != domain.StaffSuperAdmin {
		t.Fatalf("root account = %+v, want a root super admin under the normalized email", root)
	}
	if root.PasswordHash == "a-long-password" {
		t.Error("the configured password was stored in clear")
	}
	if ok, _ := uc.HasSuperAdmin(context.Background()); !ok {
		t.Error("HasSuperAdmin() = false right after creating the root")
	}
}

func TestEnsureRootAdmin_IsIdempotentAndNeverResetsAPassword(t *testing.T) {
	users := newFakeUsers()
	uc := newRootAdmin(users)
	ctx := context.Background()
	if _, err := uc.Execute(ctx, "root@betailo.test", "first-password"); err != nil {
		t.Fatal(err)
	}
	hash := users.byEmail["root@betailo.test"].PasswordHash

	// A restart with a rotated variable, even an unusable one.
	outcome, err := uc.Execute(ctx, "root@betailo.test", "x")
	if err != nil || outcome != RootAdminUnchanged {
		t.Fatalf("second Execute() = %q, %v; want unchanged", outcome, err)
	}
	if users.byEmail["root@betailo.test"].PasswordHash != hash {
		t.Error("a restart changed the root's password")
	}
}

func TestEnsureRootAdmin_PromotesAnExistingAccountAndLiftsItsSuspension(t *testing.T) {
	users := newFakeUsers()
	_ = users.Create(context.Background(), &domain.User{Email: "founder@betailo.test", PasswordHash: "hashed:own", Suspended: true})
	uc := newRootAdmin(users)

	outcome, err := uc.Execute(context.Background(), "founder@betailo.test", "")
	if err != nil || outcome != RootAdminPromoted {
		t.Fatalf("Execute() = %q, %v; want promoted (no password needed for an existing account)", outcome, err)
	}
	u := users.byEmail["founder@betailo.test"]
	if !u.Root || u.StaffRole != domain.StaffSuperAdmin || u.Suspended || u.PasswordHash != "hashed:own" {
		t.Fatalf("promoted account = %+v, want an active root that kept its password", u)
	}
}

func TestEnsureRootAdmin_ThereIsOnlyOneRoot(t *testing.T) {
	users := newFakeUsers()
	uc := newRootAdmin(users)
	ctx := context.Background()
	if _, err := uc.Execute(ctx, "first@betailo.test", "a-long-password"); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Execute(ctx, "second@betailo.test", "another-password"); err != nil {
		t.Fatal(err)
	}
	first, second := users.byEmail["first@betailo.test"], users.byEmail["second@betailo.test"]
	if first.Root || !second.Root {
		t.Fatalf("root flags = first %v, second %v; want the mark to have moved", first.Root, second.Root)
	}
	if first.StaffRole != domain.StaffSuperAdmin {
		t.Errorf("the former root's role = %q, want it to stay a super admin until someone revokes it", first.StaffRole)
	}
}

func TestEnsureRootAdmin_RefusesBadConfiguration(t *testing.T) {
	tests := []struct{ name, email, password string }{
		{"empty email", "", "a-long-password"},
		{"not an email", "root", "a-long-password"},
		{"short password for a new account", "root@betailo.test", "short"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := newFakeUsers()
			_, err := newRootAdmin(users).Execute(context.Background(), tt.email, tt.password)
			if !errors.Is(err, domain.ErrInvalid) {
				t.Fatalf("Execute() error = %v, want domain.ErrInvalid", err)
			}
			if len(users.byEmail) != 0 {
				t.Error("an account was created from a refused configuration")
			}
		})
	}
}

func TestEnsureRootAdmin_HasSuperAdminIgnoresSuspendedOnes(t *testing.T) {
	users := newFakeUsers()
	_ = users.Create(context.Background(), &domain.User{Email: "a@betailo.test", StaffRole: domain.StaffSuperAdmin, Suspended: true})
	_ = users.Create(context.Background(), &domain.User{Email: "b@betailo.test", StaffRole: domain.StaffModerator})
	if ok, _ := newRootAdmin(users).HasSuperAdmin(context.Background()); ok {
		t.Error("HasSuperAdmin() = true with only a suspended super admin and a moderator")
	}
}

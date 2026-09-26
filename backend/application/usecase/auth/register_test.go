package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pauluno777/betailo/backend/domain"
)

func TestRegister_Execute_Success(t *testing.T) {
	users := newFakeUsers()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	uc := &Register{Users: users, Hasher: fakeHasher{}, Clock: fakeClock{now: now}}

	out, err := uc.Execute(context.Background(), RegisterInput{
		Email:    " Farmer@Example.com ",
		Password: "correct-horse",
		Role:     domain.RoleFarmer,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	if out.User.Email != "farmer@example.com" {
		t.Errorf("Email = %q, want normalized %q", out.User.Email, "farmer@example.com")
	}
	if out.User.PasswordHash == "correct-horse" {
		t.Error("PasswordHash stores the plaintext password, want a hash")
	}
	if out.User.Role != domain.RoleFarmer {
		t.Errorf("Role = %v, want %v", out.User.Role, domain.RoleFarmer)
	}
	if !out.User.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", out.User.CreatedAt, now)
	}
	if out.User.ID.IsZero() {
		t.Error("ID is zero, want the repository to have assigned one")
	}
}

func TestRegister_Execute_DuplicateEmail(t *testing.T) {
	users := newFakeUsers()
	uc := &Register{Users: users, Hasher: fakeHasher{}, Clock: fakeClock{now: time.Now()}}
	ctx := context.Background()
	in := RegisterInput{Email: "buyer@example.com", Password: "correct-horse", Role: domain.RoleBuyer}

	if _, err := uc.Execute(ctx, in); err != nil {
		t.Fatalf("first Execute() error = %v, want nil", err)
	}
	_, err := uc.Execute(ctx, in)
	if !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("second Execute() error = %v, want domain.ErrEmailTaken", err)
	}
}

func TestRegister_Execute_Validation(t *testing.T) {
	tests := []struct {
		name string
		in   RegisterInput
	}{
		{"empty email", RegisterInput{Email: "", Password: "correct-horse", Role: domain.RoleFarmer}},
		{"email without @", RegisterInput{Email: "not-an-email", Password: "correct-horse", Role: domain.RoleFarmer}},
		{"password too short", RegisterInput{Email: "a@example.com", Password: "short", Role: domain.RoleFarmer}},
		{"admin role rejected", RegisterInput{Email: "a@example.com", Password: "correct-horse", Role: domain.RoleAdmin}},
		{"unknown role rejected", RegisterInput{Email: "a@example.com", Password: "correct-horse", Role: domain.Role("superuser")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &Register{Users: newFakeUsers(), Hasher: fakeHasher{}, Clock: fakeClock{now: time.Now()}}
			_, err := uc.Execute(context.Background(), tt.in)
			if !errors.Is(err, domain.ErrInvalid) {
				t.Fatalf("Execute() error = %v, want domain.ErrInvalid", err)
			}
		})
	}
}

func TestRegister_Execute_RepositoryErrorPropagates(t *testing.T) {
	wantErr := errors.New("connection reset")
	users := newFakeUsers()
	users.createErr = wantErr
	uc := &Register{Users: users, Hasher: fakeHasher{}, Clock: fakeClock{now: time.Now()}}

	_, err := uc.Execute(context.Background(), RegisterInput{
		Email: "a@example.com", Password: "correct-horse", Role: domain.RoleFarmer,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}

package security

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestBcryptHasher_RoundTrip(t *testing.T) {
	h := BcryptHasher{Cost: bcrypt.MinCost}

	hash, err := h.Hash("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if hash == "correct-horse-battery-staple" {
		t.Error("Hash() returned the plaintext password unchanged")
	}

	if err := h.Compare(hash, "correct-horse-battery-staple"); err != nil {
		t.Errorf("Compare() with the right password: error = %v, want nil", err)
	}
	if err := h.Compare(hash, "wrong-password"); err == nil {
		t.Error("Compare() with the wrong password: error = nil, want an error")
	}
}

func TestBcryptHasher_DefaultCost(t *testing.T) {
	h := BcryptHasher{} // Cost left zero

	hash, err := h.Hash("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		t.Fatalf("bcrypt.Cost() error = %v", err)
	}
	if cost != bcrypt.DefaultCost {
		t.Errorf("cost = %d, want bcrypt.DefaultCost (%d) when Cost is left zero", cost, bcrypt.DefaultCost)
	}
}

func TestBcryptHasher_DifferentHashesForSamePassword(t *testing.T) {
	h := BcryptHasher{Cost: bcrypt.MinCost}

	a, err := h.Hash("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	b, err := h.Hash("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if a == b {
		t.Error("Hash() produced identical output for two calls with the same password — salt isn't varying")
	}
}

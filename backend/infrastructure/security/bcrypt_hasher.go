// Package security implements application/ports.PasswordHasher via
// bcrypt.
package security

import (
	"golang.org/x/crypto/bcrypt"

	"github.com/pauluno777/betailo/backend/application/ports"
)

// BcryptHasher implements ports.PasswordHasher via bcrypt.
type BcryptHasher struct {
	// Cost defaults to bcrypt.DefaultCost when zero. Exposed mainly so
	// tests can use bcrypt.MinCost to keep hashing fast.
	Cost int
}

var _ ports.PasswordHasher = BcryptHasher{}

// Hash implements ports.PasswordHasher.
func (b BcryptHasher) Hash(password string) (string, error) {
	cost := b.Cost
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	out, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Compare implements ports.PasswordHasher.
func (BcryptHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

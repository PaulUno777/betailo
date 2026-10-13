// Package totp is the authenticator-app factor, on pquerna/otp.
package totp

import (
	"time"

	"github.com/pquerna/otp"
	otptotp "github.com/pquerna/otp/totp"

	"github.com/pauluno777/betailo/backend/application/ports"
)

// Service implements ports.TOTP: six digits, thirty seconds, SHA-1 (what
// every authenticator app reads), one period of clock drift tolerated.
type Service struct {
	// Issuer is the name the app shows next to the account.
	Issuer string
}

var _ ports.TOTP = (*Service)(nil)

// Generate implements ports.TOTP.
func (s *Service) Generate(account string) (string, string, error) {
	key, err := otptotp.Generate(otptotp.GenerateOpts{Issuer: s.Issuer, AccountName: account})
	if err != nil {
		return "", "", err
	}
	return key.Secret(), key.URL(), nil
}

// Validate implements ports.TOTP.
func (s *Service) Validate(secret, code string, at time.Time) bool {
	ok, err := otptotp.ValidateCustom(code, secret, at, otptotp.ValidateOpts{Period: 30, Skew: 1, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1})
	return err == nil && ok
}

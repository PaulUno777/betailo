package ports

import (
	"time"

	"github.com/pauluno777/betailo/backend/domain"
)

// TOTP is the authenticator-app factor (RFC 6238).
type TOTP interface {
	// Generate creates a secret for account and the otpauth:// URL an
	// authenticator app reads, usually as a QR code.
	Generate(account string) (secret, url string, err error)
	// Validate reports whether code is right for secret at the given time.
	Validate(secret, code string, at time.Time) bool
}

// Passkeys is the WebAuthn factor. Options and responses are the JSON
// the browser's credentials API exchanges; session is opaque state the
// caller keeps between a Begin and its Finish.
type Passkeys interface {
	BeginRegistration(u *domain.User) (options, session []byte, err error)
	// FinishRegistration returns the new passkey, without its name.
	FinishRegistration(u *domain.User, session, response []byte) (*domain.Passkey, error)
	BeginLogin(u *domain.User) (options, session []byte, err error)
	// FinishLogin returns the passkey that signed, with its counters
	// updated: the caller stores it back.
	FinishLogin(u *domain.User, session, response []byte) (*domain.Passkey, error)
}

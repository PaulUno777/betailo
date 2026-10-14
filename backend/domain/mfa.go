package domain

import "time"

// Passkey is one WebAuthn credential registered as a second factor.
type Passkey struct {
	// ID is the credential's identifier, as the authenticator chose it.
	ID []byte
	// Name is what its owner called it ("MacBook", "Clé YubiKey").
	Name      string
	CreatedAt time.Time
	// Data is the credential as the passkey service needs it back (its
	// public key and counters). Nothing else reads it.
	Data []byte
}

// MaxPasskeyNameLen bounds Passkey.Name; MaxPasskeys how many an
// account keeps.
const (
	MaxPasskeyNameLen = 60
	MaxPasskeys       = 10
)

// RecoveryCodeCount is how many single-use recovery codes an account
// holds once its authenticator app is confirmed.
const RecoveryCodeCount = 10

// MFA is an account's second factors.
type MFA struct {
	// TOTPSecret is the authenticator app's shared secret. It counts as
	// a factor only once TOTPConfirmedAt is set: the owner proved the
	// app shows the right code.
	TOTPSecret      string
	TOTPConfirmedAt *time.Time
	// RecoveryHashes are the hashes of the recovery codes still unused.
	RecoveryHashes []string
	Passkeys       []Passkey
}

// HasTOTP reports whether an authenticator app is set up and confirmed.
func (m MFA) HasTOTP() bool { return m.TOTPSecret != "" && m.TOTPConfirmedAt != nil }

// Enabled reports whether signing in asks for a second factor.
func (m MFA) Enabled() bool { return m.HasTOTP() || len(m.Passkeys) > 0 }

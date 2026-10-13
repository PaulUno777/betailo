package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// AuthTokenKind says which account flow a one-time token belongs to.
type AuthTokenKind string

const (
	// AuthTokenInvitation invites an e-mail address onto the staff.
	AuthTokenInvitation AuthTokenKind = "invitation"
	// AuthTokenPasswordReset lets its holder choose a new password.
	AuthTokenPasswordReset AuthTokenKind = "password_reset"
	// AuthTokenMFAChallenge is a sign-in whose password was right and
	// whose second factor is still owed.
	AuthTokenMFAChallenge AuthTokenKind = "mfa_challenge"
	// AuthTokenPasskeySetup holds a passkey registration in progress.
	AuthTokenPasskeySetup AuthTokenKind = "passkey_setup"
)

// How long each kind of token is good for.
const (
	InvitationTTL    = 7 * 24 * time.Hour
	PasswordResetTTL = time.Hour
	MFAChallengeTTL  = 5 * time.Minute
)

// AuthToken is one one-time token. The raw value goes to its holder (in
// a link, or back to the browser); only TokenHash is stored.
type AuthToken struct {
	ID        bson.ObjectID
	Kind      AuthTokenKind
	TokenHash string
	// UserID is the account concerned, when it exists already.
	UserID bson.ObjectID
	// Email and Role are an invitation's: who is invited, as what.
	Email string
	Role  StaffRole
	// CreatedBy is the staff member who sent an invitation.
	CreatedBy bson.ObjectID
	// Persistent carries a sign-in's "remember me" across its second step.
	Persistent bool
	// Data is opaque state of the flow (a passkey ceremony's session).
	Data      []byte
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
}

// Open reports whether the token can still be used at now.
func (t *AuthToken) Open(now time.Time) bool {
	return t.UsedAt == nil && now.Before(t.ExpiresAt)
}

package ports

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// RefreshToken is one link in a rotation chain. The raw secret handed
// to the browser (in an httpOnly cookie) is never persisted — only its
// hash, so a database read alone can never reconstruct a usable token
// (see infrastructure/jwt or wherever the hashing lives for the exact
// algorithm).
type RefreshToken struct {
	ID        bson.ObjectID
	UserID    bson.ObjectID
	FamilyID  bson.ObjectID // shared by every token issued from one login, through every rotation
	TokenHash string

	IssuedAt  time.Time
	ExpiresAt time.Time

	// Persistent carries the login's "remember me" choice forward
	// through every rotation in the family — the cookie's own Expires
	// attribute isn't re-sent by the browser on the next request, so
	// this bit has to live on the record, not be re-derived from the
	// cookie itself.
	Persistent bool

	// RevokedAt/RevokedReason/ReplacedBy are set once, never cleared.
	// RevokedReason is one of "rotated" (normal use), "logout", or
	// "reuse_detected" (see RefreshTokenRepository.RevokeFamily) —
	// kept as a plain string, not a typed enum, since nothing branches
	// on it besides a human reading the record.
	RevokedAt     *time.Time
	RevokedReason string
	ReplacedBy    *bson.ObjectID
}

// RefreshTokenRepository is the persistence port for refresh-token
// rotation chains. Deliberately narrow: no RevokeAllForUser yet — add
// it, following FarmRepository/UserRepository's whole-document-Update
// pattern, once a real caller exists (a password-change flow, or a
// "sign out everywhere" button), per this codebase's standing
// anti-premature-abstraction rule (ADR-0002).
type RefreshTokenRepository interface {
	// Create inserts the first token of a new family (at login).
	Create(ctx context.Context, t *RefreshToken) error

	// FindByHash returns domain.ErrNotFound if no token has this hash —
	// including one that has expired past Mongo's TTL cleanup, which
	// looks identical to "never existed" and is handled the same way
	// by every caller.
	FindByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)

	// Rotate atomically marks old as revoked (reason "rotated",
	// ReplacedBy set to next's eventual ID) and inserts next in the
	// same family. Implementations must guard on old not already being
	// revoked — a duplicate concurrent rotation attempt on the same
	// token must only succeed once; see
	// infrastructure/mongodb.RefreshTokenRepository.Rotate's
	// FindOneAndUpdate guard. Returns domain.ErrConflict if old is
	// already revoked by the time this call runs.
	Rotate(ctx context.Context, oldID bson.ObjectID, next *RefreshToken) error

	// RevokeFamily marks every not-yet-revoked token in familyID as
	// revoked with reason, at revokedAt (the caller's ports.Clock, not
	// server time — every other timestamp in this codebase is supplied
	// this way, so a repository never calls time.Now() itself and
	// usecases stay deterministic to test). Used both for a normal
	// logout (reason "logout") and, critically, when a revoked token is
	// presented again (reason "reuse_detected") — that second case
	// kills the whole chain, not just the reused token, since a token
	// being replayed after its rotation is the signal that it leaked.
	RevokeFamily(ctx context.Context, familyID bson.ObjectID, reason string, revokedAt time.Time) error
}

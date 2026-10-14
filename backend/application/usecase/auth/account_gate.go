package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// AccountGateTTL is how long the gate trusts a looked-up account
// status: a suspension takes effect within this delay on sessions that
// are already open, without a database read on every request.
const AccountGateTTL = 15 * time.Second

// AccountGate answers "may this signed-in account still act?" for the
// auth middleware. An access token stays cryptographically valid until
// it expires; the gate is what makes a suspension (or a deleted
// account) end sessions that are already open.
type AccountGate struct {
	Users ports.UserRepository
	Clock ports.Clock
	TTL   time.Duration

	mu    sync.Mutex
	cache map[bson.ObjectID]gateEntry
}

type gateEntry struct {
	err error
	// role is the account's staff role as stored, not as its token
	// claims: a role granted or revoked since the token was signed
	// applies on the next request.
	role    domain.StaffRole
	checked time.Time
}

// CheckActive returns nil for an active account, domain.ErrSuspended for
// a suspended one and domain.ErrUnauthorized for one that no longer
// exists. A lookup failure is returned as is and never cached.
func (g *AccountGate) CheckActive(ctx context.Context, userID bson.ObjectID) error {
	e, err := g.lookup(ctx, userID)
	if err != nil {
		return err
	}
	return e.err
}

// StaffRole returns the account's current staff role, StaffNone for an
// account that isn't staff or may no longer act. It shares CheckActive's
// cache, so the permission check behind the auth middleware costs no
// second read.
func (g *AccountGate) StaffRole(ctx context.Context, userID bson.ObjectID) (domain.StaffRole, error) {
	e, err := g.lookup(ctx, userID)
	if err != nil {
		return domain.StaffNone, err
	}
	if e.err != nil {
		return domain.StaffNone, nil
	}
	return e.role, nil
}

func (g *AccountGate) lookup(ctx context.Context, userID bson.ObjectID) (gateEntry, error) {
	ttl := g.TTL
	if ttl <= 0 {
		ttl = AccountGateTTL
	}
	now := g.Clock.Now()

	g.mu.Lock()
	if e, ok := g.cache[userID]; ok && now.Sub(e.checked) < ttl {
		g.mu.Unlock()
		return e, nil
	}
	g.mu.Unlock()

	u, err := g.Users.FindByID(ctx, userID)
	entry := gateEntry{checked: now}
	switch {
	case errors.Is(err, domain.ErrNotFound):
		entry.err = fmt.Errorf("auth: account no longer exists: %w", domain.ErrUnauthorized)
	case err != nil:
		return gateEntry{}, err
	case u.Suspended:
		entry.err = fmt.Errorf("auth: %w", domain.ErrSuspended)
	default:
		entry.role = u.StaffRole
	}

	g.mu.Lock()
	if g.cache == nil {
		g.cache = map[bson.ObjectID]gateEntry{}
	}
	// Bounded by active users; swept whenever it grows past a few
	// thousand so an idle server never keeps stale entries forever.
	if len(g.cache) > 5000 {
		for id, e := range g.cache {
			if now.Sub(e.checked) >= ttl {
				delete(g.cache, id)
			}
		}
	}
	g.cache[userID] = entry
	g.mu.Unlock()
	return entry, nil
}

// Forget drops what the gate knows about userID, so a suspension or a
// role change made in this process applies on the very next request.
func (g *AccountGate) Forget(userID bson.ObjectID) {
	g.mu.Lock()
	delete(g.cache, userID)
	g.mu.Unlock()
}

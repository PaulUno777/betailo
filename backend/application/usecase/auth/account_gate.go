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
	err     error
	checked time.Time
}

// CheckActive returns nil for an active account, domain.ErrSuspended for
// a suspended one and domain.ErrUnauthorized for one that no longer
// exists. A lookup failure is returned as is and never cached.
func (g *AccountGate) CheckActive(ctx context.Context, userID bson.ObjectID) error {
	ttl := g.TTL
	if ttl <= 0 {
		ttl = AccountGateTTL
	}
	now := g.Clock.Now()

	g.mu.Lock()
	if e, ok := g.cache[userID]; ok && now.Sub(e.checked) < ttl {
		g.mu.Unlock()
		return e.err
	}
	g.mu.Unlock()

	u, err := g.Users.FindByID(ctx, userID)
	var verdict error
	switch {
	case errors.Is(err, domain.ErrNotFound):
		verdict = fmt.Errorf("auth: account no longer exists: %w", domain.ErrUnauthorized)
	case err != nil:
		return err
	case u.Suspended:
		verdict = fmt.Errorf("auth: %w", domain.ErrSuspended)
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
	g.cache[userID] = gateEntry{err: verdict, checked: now}
	g.mu.Unlock()
	return verdict
}

// Forget drops what the gate knows about userID, so a suspension made
// in this process applies on the very next request.
func (g *AccountGate) Forget(userID bson.ObjectID) {
	g.mu.Lock()
	delete(g.cache, userID)
	g.mu.Unlock()
}

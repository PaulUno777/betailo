package ports

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

// UserRepository is the slice of user persistence usecases need.
// FindByID/Update were added once trust-score recomputation, farm-owner
// lookups (a farm's response needs its owner's trust score/verified
// badge) and admin account actions became real callers — still narrow,
// just no longer two methods.
// The auth middleware deliberately does not need this port: a verified
// JWT's claims (UserID, Role, Email) are trusted directly, keeping
// token verification stateless — no database round trip per request.
type UserRepository interface {
	// Create inserts a new user. Implementations return
	// domain.ErrEmailTaken if the email is already registered.
	Create(ctx context.Context, u *domain.User) error

	// FindByEmail returns domain.ErrNotFound if no user has this email.
	FindByEmail(ctx context.Context, email string) (*domain.User, error)

	// FindByID returns domain.ErrNotFound if no user has this ID.
	FindByID(ctx context.Context, id bson.ObjectID) (*domain.User, error)

	// List returns every registered user — the admin console's
	// "Utilisateurs" page; nothing else needs a platform-wide user scan.
	List(ctx context.Context) ([]*domain.User, error)

	// Update persists the full user document — trust score
	// recomputation, identity verification and suspension all go
	// through this one method rather than each getting a narrower
	// setter, mirroring FarmRepository.Update's whole-document replace.
	Update(ctx context.Context, u *domain.User) error
}

package ports

import (
	"context"

	"github.com/pauluno777/betailo/backend/domain"
)

// UserRepository is the slice of user persistence the auth usecases
// need. Kept to the two methods actually called (Interface
// Segregation) — widen it only when a second real caller shows up.
// The auth middleware deliberately does not need this port: a verified
// JWT's claims (UserID, Role, Email) are trusted directly, keeping
// token verification stateless — no database round trip per request.
type UserRepository interface {
	// Create inserts a new user. Implementations return
	// domain.ErrEmailTaken if the email is already registered.
	Create(ctx context.Context, u *domain.User) error

	// FindByEmail returns domain.ErrNotFound if no user has this email.
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
}

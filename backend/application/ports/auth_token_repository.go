package ports

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

// AuthTokenRepository stores the one-time tokens of account flows:
// staff invitations, password resets and the pending step of a
// two-factor sign-in. Only a token's hash is ever stored.
type AuthTokenRepository interface {
	Create(ctx context.Context, t *domain.AuthToken) error
	// FindByHash returns domain.ErrNotFound for an unknown token.
	FindByHash(ctx context.Context, kind domain.AuthTokenKind, tokenHash string) (*domain.AuthToken, error)
	FindByID(ctx context.Context, id bson.ObjectID) (*domain.AuthToken, error)
	Update(ctx context.Context, t *domain.AuthToken) error
	// ListOpen returns the tokens of a kind neither used nor expired at
	// now, newest first.
	ListOpen(ctx context.Context, kind domain.AuthTokenKind, now time.Time) ([]*domain.AuthToken, error)
}

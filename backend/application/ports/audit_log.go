package ports

import (
	"context"

	"github.com/pauluno777/betailo/backend/domain"
)

// AuditLog is the back office's journal: append-only, read newest
// first. Nothing updates or deletes a line.
type AuditLog interface {
	Record(ctx context.Context, a *domain.AdminAction) error

	// List returns the latest actions, newest first, at most limit.
	List(ctx context.Context, limit int) ([]*domain.AdminAction, error)
}

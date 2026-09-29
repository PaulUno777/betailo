package ports

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

// ReportRepository persists reports and answers the two queries the
// anti-scam flow needs: how many recent reports a target has (the
// repeated-report threshold from the architecture document's sequence
// diagram) and, for the admin console, the open ones.
type ReportRepository interface {
	Insert(ctx context.Context, r *domain.Report) error

	// FindByID returns domain.ErrNotFound if no report has this ID.
	FindByID(ctx context.Context, id bson.ObjectID) (*domain.Report, error)

	// CountRecentByTarget counts reports against targetID created at or
	// after since — the "3 signalements en 7 jours" threshold check.
	CountRecentByTarget(ctx context.Context, targetID bson.ObjectID, since time.Time) (int, error)

	// ListByTarget backs the admin detail view's "rapports reçus" list.
	ListByTarget(ctx context.Context, targetID bson.ObjectID) ([]*domain.Report, error)

	// List backs the admin console's moderation queue, filtered by
	// status (empty = all).
	List(ctx context.Context, status domain.ReportStatus) ([]*domain.Report, error)

	Update(ctx context.Context, r *domain.Report) error
}

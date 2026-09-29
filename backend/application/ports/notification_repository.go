package ports

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

// NotificationRepository is a minimal outbox: a usecase enqueues a
// notification synchronously (an unpaid reservation expiring, a report
// crossing the moderation threshold); internal/worker drains it on its
// tick. See infrastructure/mongodb's notifications collection and
// domain.Notification's doc comment for why "sent" just means "logged"
// today, not delivered through a real channel.
type NotificationRepository interface {
	Enqueue(ctx context.Context, n *domain.Notification) error
	ListPending(ctx context.Context) ([]*domain.Notification, error)
	MarkSent(ctx context.Context, id bson.ObjectID, sentAt time.Time) error
}

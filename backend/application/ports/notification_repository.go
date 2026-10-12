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
// NotificationInbox is the in-app side of notifications: what a user
// sees under the bell. Separate from the outbox port so the worker and
// the usecases that only enqueue keep a two-method dependency.
type NotificationInbox interface {
	ListByUser(ctx context.Context, userID bson.ObjectID, limit int) ([]*domain.Notification, error)
	CountUnread(ctx context.Context, userID bson.ObjectID) (int, error)
	// MarkRead returns domain.ErrNotFound unless id belongs to userID.
	MarkRead(ctx context.Context, userID, id bson.ObjectID, at time.Time) error
	MarkAllRead(ctx context.Context, userID bson.ObjectID, at time.Time) error
}

type NotificationRepository interface {
	Enqueue(ctx context.Context, n *domain.Notification) error
	ListPending(ctx context.Context) ([]*domain.Notification, error)
	MarkSent(ctx context.Context, id bson.ObjectID, sentAt time.Time) error
}

package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// NotificationStatus tracks whether internal/worker has processed a
// notification yet. There is no real email/SMS provider behind this —
// "sent" means the worker logged it — but the outbox shape (enqueue now,
// process asynchronously) is the same one a real channel would plug
// into later, per the architecture document's async-worker design.
type NotificationStatus string

const (
	NotificationPending NotificationStatus = "pending"
	NotificationSent    NotificationStatus = "sent"
)

// Notification is one outbox entry: enqueued synchronously by a usecase
// (an unpaid reservation expiring, a report crossing the moderation
// threshold), drained by internal/worker on its tick.
type Notification struct {
	ID        bson.ObjectID
	UserID    bson.ObjectID
	Type      string
	Message   string
	Status    NotificationStatus
	CreatedAt time.Time
	SentAt    *time.Time
}

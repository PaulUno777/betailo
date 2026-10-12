// Package notify is the one way usecases tell a user something
// happened: an in-app notification, also drained by the worker's
// outbox (see internal/worker).
package notify

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// Notice is one message to one user.
type Notice struct {
	UserID bson.ObjectID
	Type   string
	Params map[string]string
	Link   string
	// Fallback is the plain French text used in logs and by clients
	// that don't know Type.
	Fallback string
}

// Send enqueues n, best effort: the action that triggered it has already
// happened and must not be undone because a notification couldn't be
// written, so a failure is swallowed (the worst case is a missing bell
// entry, never a lost reservation or payment).
func Send(ctx context.Context, outbox ports.NotificationRepository, now time.Time, n Notice) {
	if outbox == nil || n.UserID.IsZero() {
		return
	}
	_ = outbox.Enqueue(ctx, &domain.Notification{
		UserID: n.UserID, Type: n.Type, Message: n.Fallback, Params: n.Params, Link: n.Link,
		Status: domain.NotificationPending, CreatedAt: now,
	})
}

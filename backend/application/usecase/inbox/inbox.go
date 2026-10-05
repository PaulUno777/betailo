// Package inbox serves a user's in-app notifications (the bell).
package inbox

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// listLimit is how much history the bell shows.
const listLimit = 50

// List returns a user's most recent notifications with the unread count.
type List struct {
	Inbox ports.NotificationInbox
}

type ListOutput struct {
	Items  []*domain.Notification
	Unread int
}

func (uc *List) Execute(ctx context.Context, userID bson.ObjectID) (*ListOutput, error) {
	items, err := uc.Inbox.ListByUser(ctx, userID, listLimit)
	if err != nil {
		return nil, err
	}
	unread, err := uc.Inbox.CountUnread(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &ListOutput{Items: items, Unread: unread}, nil
}

// UnreadCount backs the badge on the bell, polled cheaply.
type UnreadCount struct {
	Inbox ports.NotificationInbox
}

func (uc *UnreadCount) Execute(ctx context.Context, userID bson.ObjectID) (int, error) {
	return uc.Inbox.CountUnread(ctx, userID)
}

// MarkRead marks one of the user's own notifications read (ErrNotFound
// for anyone else's).
type MarkRead struct {
	Inbox ports.NotificationInbox
	Clock ports.Clock
}

func (uc *MarkRead) Execute(ctx context.Context, userID, id bson.ObjectID) error {
	return uc.Inbox.MarkRead(ctx, userID, id, uc.Clock.Now())
}

// MarkAllRead clears the badge.
type MarkAllRead struct {
	Inbox ports.NotificationInbox
	Clock ports.Clock
}

func (uc *MarkAllRead) Execute(ctx context.Context, userID bson.ObjectID) error {
	return uc.Inbox.MarkAllRead(ctx, userID, uc.Clock.Now())
}

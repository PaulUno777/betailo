package mongodb

import (
	"context"

	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// notificationDoc mirrors domain.Notification for BSON persistence.
type notificationDoc struct {
	ID        bson.ObjectID             `bson:"_id,omitempty"`
	UserID    bson.ObjectID             `bson:"userId"`
	Type      string                    `bson:"type"`
	Message   string                    `bson:"message"`
	Status    domain.NotificationStatus `bson:"status"`
	CreatedAt time.Time                 `bson:"createdAt"`
	SentAt    *time.Time                `bson:"sentAt,omitempty"`
	Params    map[string]string         `bson:"params,omitempty"`
	Link      string                    `bson:"link,omitempty"`
	ReadAt    *time.Time                `bson:"readAt,omitempty"`
}

func fromDomainNotification(n *domain.Notification) notificationDoc {
	return notificationDoc{
		ID: n.ID, UserID: n.UserID, Type: n.Type, Message: n.Message,
		Status: n.Status, CreatedAt: n.CreatedAt, SentAt: n.SentAt,
		Params: n.Params, Link: n.Link, ReadAt: n.ReadAt,
	}
}

func (d notificationDoc) toDomain() *domain.Notification {
	return &domain.Notification{
		ID: d.ID, UserID: d.UserID, Type: d.Type, Message: d.Message,
		Status: d.Status, CreatedAt: d.CreatedAt, SentAt: d.SentAt,
		Params: d.Params, Link: d.Link, ReadAt: d.ReadAt,
	}
}

// NotificationRepository implements ports.NotificationRepository
// against the "notifications" collection.
type NotificationRepository struct {
	Coll *mongo.Collection
}

var (
	_ ports.NotificationRepository = (*NotificationRepository)(nil)
	_ ports.NotificationInbox      = (*NotificationRepository)(nil)
)

// Enqueue implements ports.NotificationRepository.
func (r *NotificationRepository) Enqueue(ctx context.Context, n *domain.Notification) error {
	if n.ID.IsZero() {
		n.ID = bson.NewObjectID()
	}
	_, err := r.Coll.InsertOne(ctx, fromDomainNotification(n))
	return err
}

// ListPending implements ports.NotificationRepository.
func (r *NotificationRepository) ListPending(ctx context.Context) ([]*domain.Notification, error) {
	cur, err := r.Coll.Find(ctx, bson.M{"status": domain.NotificationPending})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []*domain.Notification
	for cur.Next(ctx) {
		var doc notificationDoc
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		out = append(out, doc.toDomain())
	}
	return out, cur.Err()
}

// MarkSent implements ports.NotificationRepository.
func (r *NotificationRepository) MarkSent(ctx context.Context, id bson.ObjectID, sentAt time.Time) error {
	_, err := r.Coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"status": domain.NotificationSent, "sentAt": sentAt,
	}})
	return err
}

// ListByUser implements ports.NotificationInbox: newest first.
func (r *NotificationRepository) ListByUser(ctx context.Context, userID bson.ObjectID, limit int) ([]*domain.Notification, error) {
	cur, err := r.Coll.Find(ctx, bson.M{"userId": userID},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	var docs []notificationDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]*domain.Notification, 0, len(docs))
	for _, d := range docs {
		out = append(out, d.toDomain())
	}
	return out, nil
}

// CountUnread implements ports.NotificationInbox.
func (r *NotificationRepository) CountUnread(ctx context.Context, userID bson.ObjectID) (int, error) {
	n, err := r.Coll.CountDocuments(ctx, bson.M{"userId": userID, "readAt": bson.M{"$exists": false}})
	return int(n), err
}

// MarkRead implements ports.NotificationInbox. Scoped to the owner, so
// nobody can mark someone else's notification read.
func (r *NotificationRepository) MarkRead(ctx context.Context, userID, id bson.ObjectID, at time.Time) error {
	res, err := r.Coll.UpdateOne(ctx, bson.M{"_id": id, "userId": userID}, bson.M{"$set": bson.M{"readAt": at}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// MarkAllRead implements ports.NotificationInbox.
func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID bson.ObjectID, at time.Time) error {
	_, err := r.Coll.UpdateMany(ctx, bson.M{"userId": userID, "readAt": bson.M{"$exists": false}}, bson.M{"$set": bson.M{"readAt": at}})
	return err
}

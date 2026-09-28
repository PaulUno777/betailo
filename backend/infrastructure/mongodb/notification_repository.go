package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

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
}

func fromDomainNotification(n *domain.Notification) notificationDoc {
	return notificationDoc{
		ID: n.ID, UserID: n.UserID, Type: n.Type, Message: n.Message,
		Status: n.Status, CreatedAt: n.CreatedAt, SentAt: n.SentAt,
	}
}

func (d notificationDoc) toDomain() *domain.Notification {
	return &domain.Notification{
		ID: d.ID, UserID: d.UserID, Type: d.Type, Message: d.Message,
		Status: d.Status, CreatedAt: d.CreatedAt, SentAt: d.SentAt,
	}
}

// NotificationRepository implements ports.NotificationRepository
// against the "notifications" collection.
type NotificationRepository struct {
	Coll *mongo.Collection
}

var _ ports.NotificationRepository = (*NotificationRepository)(nil)

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

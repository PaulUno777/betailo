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

type adminActionDoc struct {
	ID          bson.ObjectID    `bson:"_id,omitempty"`
	ActorID     bson.ObjectID    `bson:"actorId"`
	ActorName   string           `bson:"actorName,omitempty"`
	ActorRole   domain.StaffRole `bson:"actorRole,omitempty"`
	Action      string           `bson:"action"`
	TargetType  string           `bson:"targetType,omitempty"`
	TargetID    bson.ObjectID    `bson:"targetId,omitempty"`
	TargetLabel string           `bson:"targetLabel,omitempty"`
	Detail      string           `bson:"detail,omitempty"`
	CreatedAt   time.Time        `bson:"createdAt"`
}

// AuditRepository implements ports.AuditLog against the
// "admin_actions" collection.
type AuditRepository struct {
	Coll *mongo.Collection
}

var _ ports.AuditLog = (*AuditRepository)(nil)

// Record implements ports.AuditLog.
func (r *AuditRepository) Record(ctx context.Context, a *domain.AdminAction) error {
	if a.ID.IsZero() {
		a.ID = bson.NewObjectID()
	}
	_, err := r.Coll.InsertOne(ctx, adminActionDoc{
		ID: a.ID, ActorID: a.ActorID, ActorName: a.ActorName, ActorRole: a.ActorRole, Action: a.Action,
		TargetType: a.TargetType, TargetID: a.TargetID, TargetLabel: a.TargetLabel, Detail: a.Detail, CreatedAt: a.CreatedAt,
	})
	return err
}

// List implements ports.AuditLog.
func (r *AuditRepository) List(ctx context.Context, limit int) ([]*domain.AdminAction, error) {
	cur, err := r.Coll.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	var docs []adminActionDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]*domain.AdminAction, 0, len(docs))
	for _, d := range docs {
		out = append(out, &domain.AdminAction{
			ID: d.ID, ActorID: d.ActorID, ActorName: d.ActorName, ActorRole: d.ActorRole, Action: d.Action,
			TargetType: d.TargetType, TargetID: d.TargetID, TargetLabel: d.TargetLabel, Detail: d.Detail, CreatedAt: d.CreatedAt,
		})
	}
	return out, nil
}

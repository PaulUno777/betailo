package mongodb

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

type authTokenDoc struct {
	ID         bson.ObjectID        `bson:"_id,omitempty"`
	Kind       domain.AuthTokenKind `bson:"kind"`
	TokenHash  string               `bson:"tokenHash"`
	UserID     bson.ObjectID        `bson:"userId,omitempty"`
	Email      string               `bson:"email,omitempty"`
	Role       domain.StaffRole     `bson:"role,omitempty"`
	CreatedBy  bson.ObjectID        `bson:"createdBy,omitempty"`
	Persistent bool                 `bson:"persistent,omitempty"`
	Data       []byte               `bson:"data,omitempty"`
	CreatedAt  time.Time            `bson:"createdAt"`
	ExpiresAt  time.Time            `bson:"expiresAt"`
	UsedAt     *time.Time           `bson:"usedAt,omitempty"`
}

func fromDomainAuthToken(t *domain.AuthToken) authTokenDoc {
	return authTokenDoc{
		ID: t.ID, Kind: t.Kind, TokenHash: t.TokenHash, UserID: t.UserID, Email: t.Email, Role: t.Role, CreatedBy: t.CreatedBy,
		Persistent: t.Persistent, Data: t.Data, CreatedAt: t.CreatedAt, ExpiresAt: t.ExpiresAt, UsedAt: t.UsedAt,
	}
}

func (d authTokenDoc) toDomain() *domain.AuthToken {
	return &domain.AuthToken{
		ID: d.ID, Kind: d.Kind, TokenHash: d.TokenHash, UserID: d.UserID, Email: d.Email, Role: d.Role, CreatedBy: d.CreatedBy,
		Persistent: d.Persistent, Data: d.Data, CreatedAt: d.CreatedAt, ExpiresAt: d.ExpiresAt, UsedAt: d.UsedAt,
	}
}

// AuthTokenRepository implements ports.AuthTokenRepository against the
// "auth_tokens" collection.
type AuthTokenRepository struct {
	Coll *mongo.Collection
}

var _ ports.AuthTokenRepository = (*AuthTokenRepository)(nil)

// Create implements ports.AuthTokenRepository.
func (r *AuthTokenRepository) Create(ctx context.Context, t *domain.AuthToken) error {
	if t.ID.IsZero() {
		t.ID = bson.NewObjectID()
	}
	_, err := r.Coll.InsertOne(ctx, fromDomainAuthToken(t))
	return err
}

func (r *AuthTokenRepository) findOne(ctx context.Context, filter bson.M) (*domain.AuthToken, error) {
	var doc authTokenDoc
	err := r.Coll.FindOne(ctx, filter).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return doc.toDomain(), nil
}

// FindByHash implements ports.AuthTokenRepository.
func (r *AuthTokenRepository) FindByHash(ctx context.Context, kind domain.AuthTokenKind, tokenHash string) (*domain.AuthToken, error) {
	return r.findOne(ctx, bson.M{"kind": kind, "tokenHash": tokenHash})
}

// FindByID implements ports.AuthTokenRepository.
func (r *AuthTokenRepository) FindByID(ctx context.Context, id bson.ObjectID) (*domain.AuthToken, error) {
	return r.findOne(ctx, bson.M{"_id": id})
}

// Update implements ports.AuthTokenRepository.
func (r *AuthTokenRepository) Update(ctx context.Context, t *domain.AuthToken) error {
	res, err := r.Coll.ReplaceOne(ctx, bson.M{"_id": t.ID}, fromDomainAuthToken(t))
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ListOpen implements ports.AuthTokenRepository.
func (r *AuthTokenRepository) ListOpen(ctx context.Context, kind domain.AuthTokenKind, now time.Time) ([]*domain.AuthToken, error) {
	cur, err := r.Coll.Find(ctx,
		bson.M{"kind": kind, "usedAt": bson.M{"$exists": false}, "expiresAt": bson.M{"$gt": now}},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	var docs []authTokenDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]*domain.AuthToken, 0, len(docs))
	for _, d := range docs {
		out = append(out, d.toDomain())
	}
	return out, nil
}

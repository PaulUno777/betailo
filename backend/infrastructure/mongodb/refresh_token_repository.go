package mongodb

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// refreshTokenDoc mirrors ports.RefreshToken for BSON persistence —
// bson tags live only here, never on the port type, matching
// userDoc/domain.User's split.
type refreshTokenDoc struct {
	ID            bson.ObjectID  `bson:"_id,omitempty"`
	UserID        bson.ObjectID  `bson:"userId"`
	FamilyID      bson.ObjectID  `bson:"familyId"`
	TokenHash     string         `bson:"tokenHash"`
	IssuedAt      time.Time      `bson:"issuedAt"`
	ExpiresAt     time.Time      `bson:"expiresAt"`
	Persistent    bool           `bson:"persistent"`
	RevokedAt     *time.Time     `bson:"revokedAt,omitempty"`
	RevokedReason string         `bson:"revokedReason,omitempty"`
	ReplacedBy    *bson.ObjectID `bson:"replacedBy,omitempty"`
}

func fromPortRefreshToken(t *ports.RefreshToken) refreshTokenDoc {
	return refreshTokenDoc{
		ID:            t.ID,
		UserID:        t.UserID,
		FamilyID:      t.FamilyID,
		TokenHash:     t.TokenHash,
		IssuedAt:      t.IssuedAt,
		ExpiresAt:     t.ExpiresAt,
		Persistent:    t.Persistent,
		RevokedAt:     t.RevokedAt,
		RevokedReason: t.RevokedReason,
		ReplacedBy:    t.ReplacedBy,
	}
}

func (d refreshTokenDoc) toPort() *ports.RefreshToken {
	return &ports.RefreshToken{
		ID:            d.ID,
		UserID:        d.UserID,
		FamilyID:      d.FamilyID,
		TokenHash:     d.TokenHash,
		IssuedAt:      d.IssuedAt,
		ExpiresAt:     d.ExpiresAt,
		Persistent:    d.Persistent,
		RevokedAt:     d.RevokedAt,
		RevokedReason: d.RevokedReason,
		ReplacedBy:    d.ReplacedBy,
	}
}

// RefreshTokenRepository implements ports.RefreshTokenRepository
// against the "refresh_tokens" collection. Expired documents are
// cleaned up by Mongo's own TTL index (see ensureIndexes) — this type
// never deletes on expiry itself.
type RefreshTokenRepository struct {
	Coll *mongo.Collection
}

var _ ports.RefreshTokenRepository = (*RefreshTokenRepository)(nil)

// Create implements ports.RefreshTokenRepository.
func (r *RefreshTokenRepository) Create(ctx context.Context, t *ports.RefreshToken) error {
	if t.ID.IsZero() {
		t.ID = bson.NewObjectID()
	}
	_, err := r.Coll.InsertOne(ctx, fromPortRefreshToken(t))
	return err
}

// FindByHash implements ports.RefreshTokenRepository.
func (r *RefreshTokenRepository) FindByHash(ctx context.Context, tokenHash string) (*ports.RefreshToken, error) {
	var doc refreshTokenDoc
	err := r.Coll.FindOne(ctx, bson.M{"tokenHash": tokenHash}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return doc.toPort(), nil
}

// Rotate implements ports.RefreshTokenRepository. The update is guarded
// on revokedAt not already being set, so a duplicate concurrent
// rotation attempt on the same token (e.g. a retried request) only
// ever succeeds once — the loser sees MatchedCount 0 and reports
// domain.ErrConflict rather than silently inserting a second
// "next" token into the family. This is intentionally not wrapped in
// TxManager's multi-document transaction: a crash between the update
// and the insert just forces that one user to log in again, not a
// financial-integrity bug worth transaction overhead for.
func (r *RefreshTokenRepository) Rotate(ctx context.Context, oldID bson.ObjectID, next *ports.RefreshToken) error {
	if next.ID.IsZero() {
		next.ID = bson.NewObjectID()
	}

	res, err := r.Coll.UpdateOne(ctx,
		bson.M{"_id": oldID, "revokedAt": nil},
		bson.M{"$set": bson.M{
			"revokedAt":     next.IssuedAt,
			"revokedReason": "rotated",
			"replacedBy":    next.ID,
		}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return domain.ErrConflict
	}

	_, err = r.Coll.InsertOne(ctx, fromPortRefreshToken(next))
	return err
}

// RevokeFamily implements ports.RefreshTokenRepository. UpdateMany
// rather than a single document: a family can have more than one
// not-yet-revoked token only in the reuse-detected case this method
// exists to handle, but it's written to be correct regardless of how
// many match.
func (r *RefreshTokenRepository) RevokeFamily(ctx context.Context, familyID bson.ObjectID, reason string, revokedAt time.Time) error {
	_, err := r.Coll.UpdateMany(ctx,
		bson.M{"familyId": familyID, "revokedAt": nil},
		bson.M{"$set": bson.M{"revokedAt": revokedAt, "revokedReason": reason}},
	)
	return err
}

// RevokeAllForUser implements ports.RefreshTokenRepository.
func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID bson.ObjectID, reason string, revokedAt time.Time) error {
	_, err := r.Coll.UpdateMany(ctx,
		bson.M{"userId": userID, "revokedAt": nil},
		bson.M{"$set": bson.M{"revokedAt": revokedAt, "revokedReason": reason}},
	)
	return err
}

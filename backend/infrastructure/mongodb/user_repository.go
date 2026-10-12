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

// duplicateKeyCode is the MongoDB server error code for a unique-index
// violation — what users.email's unique index (see
// mongodb.ensureIndexes) returns when Create races another
// registration for the same email.
const duplicateKeyCode = 11000

// userDoc mirrors domain.User for BSON persistence. domain.User itself
// carries no bson tags (see domain/user.go); mapping happens only here,
// at the infrastructure boundary.
type userDoc struct {
	ID           bson.ObjectID `bson:"_id,omitempty"`
	Email        string        `bson:"email"`
	PasswordHash string        `bson:"passwordHash"`
	Role         domain.Role   `bson:"role"`
	DisplayName  string        `bson:"displayName,omitempty"`
	Phone        string        `bson:"phone,omitempty"`
	City         string        `bson:"city,omitempty"`
	IsVerified   bool          `bson:"isVerified"`
	Suspended    bool          `bson:"suspended,omitempty"`
	TrustScore   float64       `bson:"trustScore"`
	CreatedAt    time.Time     `bson:"createdAt"`
}

func fromDomainUser(u *domain.User) userDoc {
	return userDoc{
		ID:           u.ID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         u.Role,
		DisplayName:  u.DisplayName,
		Phone:        u.Phone,
		City:         u.City,
		IsVerified:   u.IsVerified,
		Suspended:    u.Suspended,
		TrustScore:   u.TrustScore,
		CreatedAt:    u.CreatedAt,
	}
}

func (d userDoc) toDomain() *domain.User {
	return &domain.User{
		ID:           d.ID,
		Email:        d.Email,
		PasswordHash: d.PasswordHash,
		Role:         d.Role,
		DisplayName:  d.DisplayName,
		Phone:        d.Phone,
		City:         d.City,
		IsVerified:   d.IsVerified,
		Suspended:    d.Suspended,
		TrustScore:   d.TrustScore,
		CreatedAt:    d.CreatedAt,
	}
}

// UserRepository implements ports.UserRepository against the "users"
// collection.
type UserRepository struct {
	Coll *mongo.Collection
}

var _ ports.UserRepository = (*UserRepository)(nil)

// Create implements ports.UserRepository. It assigns u.ID before
// inserting and returns domain.ErrEmailTaken if users.email's unique
// index rejects the insert — see
// application/usecase/auth.Register.Execute for why this is the only
// place the "email already taken" check happens.
func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	if u.ID.IsZero() {
		u.ID = bson.NewObjectID()
	}

	_, err := r.Coll.InsertOne(ctx, fromDomainUser(u))
	if err == nil {
		return nil
	}

	var writeErr mongo.WriteException
	if errors.As(err, &writeErr) && writeErr.HasErrorCode(duplicateKeyCode) {
		return domain.ErrEmailTaken
	}
	return err
}

// FindByEmail implements ports.UserRepository.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var doc userDoc
	err := r.Coll.FindOne(ctx, bson.M{"email": email}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return doc.toDomain(), nil
}

// FindByID implements ports.UserRepository.
func (r *UserRepository) FindByID(ctx context.Context, id bson.ObjectID) (*domain.User, error) {
	var doc userDoc
	err := r.Coll.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return doc.toDomain(), nil
}

// List implements ports.UserRepository.
func (r *UserRepository) List(ctx context.Context) ([]*domain.User, error) {
	cur, err := r.Coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []*domain.User
	for cur.Next(ctx) {
		var doc userDoc
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		out = append(out, doc.toDomain())
	}
	return out, cur.Err()
}

// Update implements ports.UserRepository.
func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	res, err := r.Coll.ReplaceOne(ctx, bson.M{"_id": u.ID}, fromDomainUser(u))
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

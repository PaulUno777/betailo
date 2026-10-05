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

// farmDoc mirrors domain.Farm for BSON persistence. domain.Farm itself
// carries no bson tags (see domain/farm.go); mapping happens only here,
// at the infrastructure boundary.
type farmDoc struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	OwnerID   bson.ObjectID `bson:"ownerId"`
	Name      string        `bson:"name"`
	Location  string        `bson:"location"`
	Species   []string      `bson:"species,omitempty"`
	CreatedAt time.Time     `bson:"createdAt"`

	Description  string `bson:"description,omitempty"`
	ContactPhone string `bson:"contactPhone,omitempty"`
}

func fromDomainFarm(f *domain.Farm) farmDoc {
	return farmDoc{
		ID:        f.ID,
		OwnerID:   f.OwnerID,
		Name:      f.Name,
		Location:  f.Location,
		Species:   f.Species,
		CreatedAt: f.CreatedAt,

		Description:  f.Description,
		ContactPhone: f.ContactPhone,
	}
}

func (d farmDoc) toDomain() *domain.Farm {
	return &domain.Farm{
		ID:        d.ID,
		OwnerID:   d.OwnerID,
		Name:      d.Name,
		Location:  d.Location,
		Species:   d.Species,
		CreatedAt: d.CreatedAt,

		Description:  d.Description,
		ContactPhone: d.ContactPhone,
	}
}

// FarmRepository implements ports.FarmRepository against the "farms"
// collection.
type FarmRepository struct {
	Coll *mongo.Collection
}

var _ ports.FarmRepository = (*FarmRepository)(nil)

// Create implements ports.FarmRepository.
func (r *FarmRepository) Create(ctx context.Context, f *domain.Farm) error {
	if f.ID.IsZero() {
		f.ID = bson.NewObjectID()
	}
	_, err := r.Coll.InsertOne(ctx, fromDomainFarm(f))
	return err
}

// FindByID implements ports.FarmRepository.
func (r *FarmRepository) FindByID(ctx context.Context, id bson.ObjectID) (*domain.Farm, error) {
	var doc farmDoc
	err := r.Coll.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return doc.toDomain(), nil
}

// ListByOwner implements ports.FarmRepository.
func (r *FarmRepository) ListByOwner(ctx context.Context, ownerID bson.ObjectID) ([]*domain.Farm, error) {
	cur, err := r.Coll.Find(ctx, bson.M{"ownerId": ownerID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []*domain.Farm
	for cur.Next(ctx) {
		var doc farmDoc
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		out = append(out, doc.toDomain())
	}
	return out, cur.Err()
}

// Update implements ports.FarmRepository.
func (r *FarmRepository) Update(ctx context.Context, f *domain.Farm) error {
	res, err := r.Coll.ReplaceOne(ctx, bson.M{"_id": f.ID}, fromDomainFarm(f))
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Delete implements ports.FarmRepository.
func (r *FarmRepository) Delete(ctx context.Context, id bson.ObjectID) error {
	res, err := r.Coll.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

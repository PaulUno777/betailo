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

// farmDoc mirrors domain.Farm for BSON persistence. domain.Farm itself
// carries no bson tags (see domain/farm.go); mapping happens only here,
// at the infrastructure boundary.
type farmDoc struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	OwnerID   bson.ObjectID `bson:"ownerId"`
	Name      string        `bson:"name"`
	Location  string        `bson:"location"`
	Country   string        `bson:"country,omitempty"`
	Region    string        `bson:"region,omitempty"`
	Species   []string      `bson:"species,omitempty"`
	CreatedAt time.Time     `bson:"createdAt"`

	Description  string `bson:"description,omitempty"`
	ContactPhone string `bson:"contactPhone,omitempty"`

	Status       domain.FarmStatus   `bson:"status"`
	Verification farmVerificationDoc `bson:"verification,omitempty"`
}

type farmVerificationDoc struct {
	IDDocumentKey   string        `bson:"idDocumentKey,omitempty"`
	PhotoKeys       []string      `bson:"photoKeys,omitempty"`
	SubmittedAt     *time.Time    `bson:"submittedAt,omitempty"`
	ReviewedAt      *time.Time    `bson:"reviewedAt,omitempty"`
	ReviewedBy      bson.ObjectID `bson:"reviewedBy,omitempty"`
	AutoApproved    bool          `bson:"autoApproved,omitempty"`
	RejectionReason string        `bson:"rejectionReason,omitempty"`
}

func fromDomainFarm(f *domain.Farm) farmDoc {
	return farmDoc{
		ID:        f.ID,
		OwnerID:   f.OwnerID,
		Name:      f.Name,
		Location:  f.Location,
		Country:   f.Country,
		Region:    f.Region,
		Species:   f.Species,
		CreatedAt: f.CreatedAt,

		Description:  f.Description,
		ContactPhone: f.ContactPhone,

		Status: f.Status,
		Verification: farmVerificationDoc{
			IDDocumentKey: f.Verification.IDDocumentKey, PhotoKeys: f.Verification.PhotoKeys,
			SubmittedAt: f.Verification.SubmittedAt, ReviewedAt: f.Verification.ReviewedAt, ReviewedBy: f.Verification.ReviewedBy,
			AutoApproved: f.Verification.AutoApproved, RejectionReason: f.Verification.RejectionReason,
		},
	}
}

func (d farmDoc) toDomain() *domain.Farm {
	return &domain.Farm{
		ID:        d.ID,
		OwnerID:   d.OwnerID,
		Name:      d.Name,
		Location:  d.Location,
		Country:   d.Country,
		Region:    d.Region,
		Species:   d.Species,
		CreatedAt: d.CreatedAt,

		Description:  d.Description,
		ContactPhone: d.ContactPhone,

		Status: d.Status,
		Verification: domain.FarmVerification{
			IDDocumentKey: d.Verification.IDDocumentKey, PhotoKeys: d.Verification.PhotoKeys,
			SubmittedAt: d.Verification.SubmittedAt, ReviewedAt: d.Verification.ReviewedAt, ReviewedBy: d.Verification.ReviewedBy,
			AutoApproved: d.Verification.AutoApproved, RejectionReason: d.Verification.RejectionReason,
		},
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

// ListByStatus implements ports.FarmRepository.
func (r *FarmRepository) ListByStatus(ctx context.Context, status domain.FarmStatus) ([]*domain.Farm, error) {
	filter := bson.M{}
	if status != "" {
		filter["status"] = status
	}
	cur, err := r.Coll.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "verification.submittedAt", Value: 1}, {Key: "createdAt", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var docs []farmDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]*domain.Farm, 0, len(docs))
	for _, d := range docs {
		out = append(out, d.toDomain())
	}
	return out, nil
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

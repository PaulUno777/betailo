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
	Reference string        `bson:"reference,omitempty"`
	OwnerID   bson.ObjectID `bson:"ownerId"`
	Name      string        `bson:"name"`
	Location  string        `bson:"location"`
	Country   string        `bson:"country,omitempty"`
	Region    string        `bson:"region,omitempty"`
	Species   []string      `bson:"species,omitempty"`
	CreatedAt time.Time     `bson:"createdAt"`

	Description string           `bson:"description,omitempty"`
	Contacts    []farmContactDoc `bson:"contacts,omitempty"`
	LogoKey     string           `bson:"logoKey,omitempty"`
	BannerKey   string           `bson:"bannerKey,omitempty"`

	Status       domain.FarmStatus   `bson:"status"`
	Verification farmVerificationDoc `bson:"verification,omitempty"`
}

type farmContactDoc struct {
	Phone    string `bson:"phone"`
	WhatsApp bool   `bson:"whatsapp"`
}

type farmVerificationDoc struct {
	IDDocumentType    domain.IDDocumentType `bson:"idDocumentType,omitempty"`
	IDDocumentKey     string                `bson:"idDocumentKey,omitempty"`
	IDDocumentBackKey string                `bson:"idDocumentBackKey,omitempty"`
	PhotoKeys         []string              `bson:"photoKeys,omitempty"`
	SubmittedAt       *time.Time            `bson:"submittedAt,omitempty"`
	Submissions       int                   `bson:"submissions,omitempty"`
	ReviewedAt        *time.Time            `bson:"reviewedAt,omitempty"`
	ReviewedBy        bson.ObjectID         `bson:"reviewedBy,omitempty"`
	ReviewedByRole    domain.StaffRole      `bson:"reviewedByRole,omitempty"`
	AutoApproved      bool                  `bson:"autoApproved,omitempty"`
	RejectionCode     domain.RejectionCode  `bson:"rejectionCode,omitempty"`
	RejectionReason   string                `bson:"rejectionReason,omitempty"`
	Checks            []string              `bson:"checks,omitempty"`
}

func fromDomainFarm(f *domain.Farm) farmDoc {
	return farmDoc{
		ID:        f.ID,
		Reference: f.Reference,
		OwnerID:   f.OwnerID,
		Name:      f.Name,
		Location:  f.Location,
		Country:   f.Country,
		Region:    f.Region,
		Species:   f.Species,
		CreatedAt: f.CreatedAt,

		Description: f.Description,
		Contacts:    contactDocs(f.Contacts),
		LogoKey:     f.LogoKey,
		BannerKey:   f.BannerKey,

		Status: f.Status,
		Verification: farmVerificationDoc{
			IDDocumentType: f.Verification.IDDocumentType, IDDocumentKey: f.Verification.IDDocumentKey,
			IDDocumentBackKey: f.Verification.IDDocumentBackKey, PhotoKeys: f.Verification.PhotoKeys,
			SubmittedAt: f.Verification.SubmittedAt, Submissions: f.Verification.Submissions,
			ReviewedAt: f.Verification.ReviewedAt, ReviewedBy: f.Verification.ReviewedBy, ReviewedByRole: f.Verification.ReviewedByRole,
			AutoApproved: f.Verification.AutoApproved, RejectionCode: f.Verification.RejectionCode,
			RejectionReason: f.Verification.RejectionReason, Checks: f.Verification.Checks,
		},
	}
}

func contactDocs(in []domain.FarmContact) []farmContactDoc {
	var out []farmContactDoc
	for _, c := range in {
		out = append(out, farmContactDoc{Phone: c.Phone, WhatsApp: c.WhatsApp})
	}
	return out
}

func domainContacts(in []farmContactDoc) []domain.FarmContact {
	var out []domain.FarmContact
	for _, c := range in {
		out = append(out, domain.FarmContact{Phone: c.Phone, WhatsApp: c.WhatsApp})
	}
	return out
}

func (d farmDoc) toDomain() *domain.Farm {
	return &domain.Farm{
		ID:        d.ID,
		Reference: d.Reference,
		OwnerID:   d.OwnerID,
		Name:      d.Name,
		Location:  d.Location,
		Country:   d.Country,
		Region:    d.Region,
		Species:   d.Species,
		CreatedAt: d.CreatedAt,

		Description: d.Description,
		Contacts:    domainContacts(d.Contacts),
		LogoKey:     d.LogoKey,
		BannerKey:   d.BannerKey,

		Status: d.Status,
		Verification: domain.FarmVerification{
			IDDocumentType: d.Verification.IDDocumentType, IDDocumentKey: d.Verification.IDDocumentKey,
			IDDocumentBackKey: d.Verification.IDDocumentBackKey, PhotoKeys: d.Verification.PhotoKeys,
			SubmittedAt: d.Verification.SubmittedAt, Submissions: d.Verification.Submissions,
			ReviewedAt: d.Verification.ReviewedAt, ReviewedBy: d.Verification.ReviewedBy, ReviewedByRole: d.Verification.ReviewedByRole,
			AutoApproved: d.Verification.AutoApproved, RejectionCode: d.Verification.RejectionCode,
			RejectionReason: d.Verification.RejectionReason, Checks: d.Verification.Checks,
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

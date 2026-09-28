package mongodb

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// listingRefDoc is the minimal "listings" projection reservation reads
// — kept separate from listingDoc below so that narrow, cross-module
// read never has to change shape when the full document does.
type listingRefDoc struct {
	CycleID   *bson.ObjectID `bson:"cycleId"`
	UnitPrice float64        `bson:"unitPrice"`
}

// listingDoc mirrors domain.Listing for BSON persistence. domain.Listing
// itself carries no bson tags (see domain/listing.go); mapping happens
// only here, at the infrastructure boundary.
type listingDoc struct {
	ID          bson.ObjectID        `bson:"_id,omitempty"`
	CycleID     *bson.ObjectID       `bson:"cycleId,omitempty"`
	Title       string               `bson:"title"`
	Description string               `bson:"description,omitempty"`
	UnitPrice   float64              `bson:"unitPrice"`
	Unit        string               `bson:"unit,omitempty"`
	Status      domain.ListingStatus `bson:"status"`
	Photos      []string             `bson:"photos,omitempty"`
	Flagged     bool                 `bson:"flagged,omitempty"`
}

func fromDomainListing(l *domain.Listing) listingDoc {
	return listingDoc{
		ID:          l.ID,
		CycleID:     l.CycleID,
		Title:       l.Title,
		Description: l.Description,
		UnitPrice:   l.UnitPrice,
		Unit:        l.Unit,
		Status:      l.Status,
		Photos:      l.Photos,
		Flagged:     l.Flagged,
	}
}

func (d listingDoc) toDomain() *domain.Listing {
	return &domain.Listing{
		ID:          d.ID,
		CycleID:     d.CycleID,
		Title:       d.Title,
		Description: d.Description,
		UnitPrice:   d.UnitPrice,
		Unit:        d.Unit,
		Status:      d.Status,
		Photos:      d.Photos,
		Flagged:     d.Flagged,
	}
}

// ListingRepository implements ports.ListingRepository (reservation's
// narrow slice) and ports.ListingStore (the fuller CRUD port
// application/usecase/listing depends on) against the "listings"
// collection — see ports.ListingStore's doc comment for why one
// concrete type implements two interfaces here. It does not depend on
// any other module's repository, so reservation and listing stay
// decoupled at the module boundary — modules don't call directly into
// each other's services, per the architecture document.
type ListingRepository struct {
	Coll *mongo.Collection
}

var (
	_ ports.ListingRepository = (*ListingRepository)(nil)
	_ ports.ListingStore      = (*ListingRepository)(nil)
)

// CycleAndPrice implements ports.ListingRepository.
func (r *ListingRepository) CycleAndPrice(ctx context.Context, listingID bson.ObjectID) (bson.ObjectID, float64, error) {
	var doc listingRefDoc
	if err := r.Coll.FindOne(ctx, bson.M{"_id": listingID}).Decode(&doc); err != nil {
		return bson.ObjectID{}, 0, err
	}
	if doc.CycleID == nil {
		return bson.ObjectID{}, 0, errors.New("mongodb: listing has no cycle")
	}
	return *doc.CycleID, doc.UnitPrice, nil
}

// Create implements ports.ListingStore.
func (r *ListingRepository) Create(ctx context.Context, l *domain.Listing) error {
	if l.ID.IsZero() {
		l.ID = bson.NewObjectID()
	}
	_, err := r.Coll.InsertOne(ctx, fromDomainListing(l))
	return err
}

// FindByID implements ports.ListingStore.
func (r *ListingRepository) FindByID(ctx context.Context, id bson.ObjectID) (*domain.Listing, error) {
	var doc listingDoc
	err := r.Coll.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return doc.toDomain(), nil
}

// FindByCycleID implements ports.ListingStore.
func (r *ListingRepository) FindByCycleID(ctx context.Context, cycleID bson.ObjectID) (*domain.Listing, error) {
	var doc listingDoc
	err := r.Coll.FindOne(ctx, bson.M{"cycleId": cycleID}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return doc.toDomain(), nil
}

// Update implements ports.ListingStore.
func (r *ListingRepository) Update(ctx context.Context, l *domain.Listing) error {
	res, err := r.Coll.ReplaceOne(ctx, bson.M{"_id": l.ID}, fromDomainListing(l))
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

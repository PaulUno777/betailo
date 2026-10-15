package mongodb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// CatalogRepository implements ports.CatalogRepository against
// "catalog_versions": one document per version, its number as _id (so
// two publishes can't take the same one), its lists kept as the JSON
// that was published.
type CatalogRepository struct {
	Coll *mongo.Collection
}

var _ ports.CatalogRepository = (*CatalogRepository)(nil)

type catalogDoc struct {
	Version     int           `bson:"_id"`
	Note        string        `bson:"note,omitempty"`
	PublishedBy bson.ObjectID `bson:"publishedBy,omitempty"`
	PublishedAt time.Time     `bson:"publishedAt"`
	Content     string        `bson:"content"`
}

func (d *catalogDoc) toDomain() (*domain.Catalog, error) {
	var c domain.Catalog
	if err := json.Unmarshal([]byte(d.Content), &c); err != nil {
		return nil, fmt.Errorf("catalog version %d: %w", d.Version, err)
	}
	c.Version, c.Note, c.PublishedBy, c.PublishedAt = d.Version, d.Note, d.PublishedBy, d.PublishedAt
	return &c, nil
}

// Latest implements ports.CatalogRepository.
func (r *CatalogRepository) Latest(ctx context.Context) (*domain.Catalog, error) {
	var d catalogDoc
	err := r.Coll.FindOne(ctx, bson.M{}, options.FindOne().SetSort(bson.D{{Key: "_id", Value: -1}})).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return d.toDomain()
}

// Get implements ports.CatalogRepository.
func (r *CatalogRepository) Get(ctx context.Context, version int) (*domain.Catalog, error) {
	var d catalogDoc
	err := r.Coll.FindOne(ctx, bson.M{"_id": version}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return d.toDomain()
}

// List implements ports.CatalogRepository.
func (r *CatalogRepository) List(ctx context.Context) ([]*domain.Catalog, error) {
	cur, err := r.Coll.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "_id", Value: -1}}))
	if err != nil {
		return nil, err
	}
	var docs []catalogDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]*domain.Catalog, 0, len(docs))
	for i := range docs {
		c, err := docs[i].toDomain()
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// Insert implements ports.CatalogRepository.
func (r *CatalogRepository) Insert(ctx context.Context, c *domain.Catalog) error {
	lists := *c
	lists.Version, lists.Note, lists.PublishedAt = 0, "", time.Time{}
	content, err := json.Marshal(catalogLists(lists))
	if err != nil {
		return err
	}
	_, err = r.Coll.InsertOne(ctx, catalogDoc{
		Version: c.Version, Note: c.Note, PublishedBy: c.PublishedBy, PublishedAt: c.PublishedAt, Content: string(content),
	})
	if mongo.IsDuplicateKeyError(err) {
		return domain.Coded("catalog.version_taken", fmt.Errorf("catalog: version %d was just published by someone else: %w", c.Version, domain.ErrConflict))
	}
	return err
}

// catalogLists is a Catalog without its version metadata: what the
// content field holds.
type catalogLists domain.Catalog

func (l catalogLists) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Categories       []domain.CatalogCategory `json:"categories"`
		Countries        []domain.CatalogCountry  `json:"countries"`
		ForumCategories  []domain.CatalogTerm     `json:"forumCategories"`
		ReviewTags       []domain.CatalogTerm     `json:"reviewTags"`
		ReportReasons    []domain.CatalogTerm     `json:"reportReasons"`
		FarmReviewChecks []domain.CatalogTerm     `json:"farmReviewChecks"`
	}{l.Categories, l.Countries, l.ForumCategories, l.ReviewTags, l.ReportReasons, l.FarmReviewChecks})
}

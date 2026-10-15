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

// LegacyCatalogRepository reads "catalog_versions", where the reference
// data was kept as one document per version before the references were
// split. Only the one-time move to ReferenceRepository reads it.
type LegacyCatalogRepository struct {
	Coll *mongo.Collection
}

var _ ports.LegacyCatalogSource = (*LegacyCatalogRepository)(nil)

type legacyCatalogDoc struct {
	Version      int    `bson:"_id"`
	SeedRevision int    `bson:"seedRevision,omitempty"`
	Content      string `bson:"content"`
}

// Latest implements ports.LegacyCatalogSource.
func (r *LegacyCatalogRepository) Latest(ctx context.Context) (*domain.Catalog, int, error) {
	var d legacyCatalogDoc
	err := r.Coll.FindOne(ctx, bson.M{}, options.FindOne().SetSort(bson.D{{Key: "_id", Value: -1}})).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, 0, domain.ErrNotFound
	}
	if err != nil {
		return nil, 0, err
	}
	var c domain.Catalog
	if err := json.Unmarshal([]byte(d.Content), &c); err != nil {
		return nil, 0, fmt.Errorf("catalog version %d: %w", d.Version, err)
	}
	return &c, d.SeedRevision, nil
}

// ReferenceRepository implements ports.CatalogRepository against
// "reference_versions": one document per version of each reference,
// keyed "<reference>:<version>" (so two publishes can't take the same
// number), its content kept as the JSON that was published.
type ReferenceRepository struct {
	Coll *mongo.Collection
}

var _ ports.CatalogRepository = (*ReferenceRepository)(nil)

type referenceDoc struct {
	ID           string        `bson:"_id"`
	Name         string        `bson:"name"`
	Version      int           `bson:"version"`
	Note         string        `bson:"note,omitempty"`
	PublishedBy  bson.ObjectID `bson:"publishedBy,omitempty"`
	PublishedAt  time.Time     `bson:"publishedAt"`
	SeedRevision int           `bson:"seedRevision,omitempty"`
	Content      string        `bson:"content"`
}

func (d *referenceDoc) toDomain() *domain.CatalogPart {
	return &domain.CatalogPart{
		Name: d.Name, Version: d.Version, Note: d.Note, PublishedBy: d.PublishedBy, PublishedAt: d.PublishedAt,
		SeedRevision: d.SeedRevision, Content: json.RawMessage(d.Content),
	}
}

func (r *ReferenceRepository) one(ctx context.Context, filter bson.M, opts ...options.Lister[options.FindOneOptions]) (*domain.CatalogPart, error) {
	var d referenceDoc
	err := r.Coll.FindOne(ctx, filter, opts...).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return d.toDomain(), nil
}

// Latest implements ports.CatalogRepository.
func (r *ReferenceRepository) Latest(ctx context.Context, name string) (*domain.CatalogPart, error) {
	return r.one(ctx, bson.M{"name": name}, options.FindOne().SetSort(bson.D{{Key: "version", Value: -1}}))
}

// Get implements ports.CatalogRepository.
func (r *ReferenceRepository) Get(ctx context.Context, name string, version int) (*domain.CatalogPart, error) {
	return r.one(ctx, bson.M{"name": name, "version": version})
}

// List implements ports.CatalogRepository.
func (r *ReferenceRepository) List(ctx context.Context, name string) ([]*domain.CatalogPart, error) {
	filter := bson.M{}
	if name != "" {
		filter["name"] = name
	}
	cur, err := r.Coll.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "publishedAt", Value: -1}, {Key: "version", Value: -1}}))
	if err != nil {
		return nil, err
	}
	var docs []referenceDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]*domain.CatalogPart, 0, len(docs))
	for i := range docs {
		out = append(out, docs[i].toDomain())
	}
	return out, nil
}

// Insert implements ports.CatalogRepository.
func (r *ReferenceRepository) Insert(ctx context.Context, p *domain.CatalogPart) error {
	_, err := r.Coll.InsertOne(ctx, referenceDoc{
		ID: fmt.Sprintf("%s:%d", p.Name, p.Version), Name: p.Name, Version: p.Version, Note: p.Note,
		PublishedBy: p.PublishedBy, PublishedAt: p.PublishedAt, SeedRevision: p.SeedRevision, Content: string(p.Content),
	})
	if mongo.IsDuplicateKeyError(err) {
		return domain.CodedWith("catalog.version_taken", map[string]any{"reference": p.Name},
			fmt.Errorf("catalog: %s version %d was just published by someone else: %w", p.Name, p.Version, domain.ErrConflict))
	}
	return err
}

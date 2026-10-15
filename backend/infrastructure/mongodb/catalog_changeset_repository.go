package mongodb

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// changesetID is the one shared changeset's _id.
const changesetID = "pending"

// CatalogChangesetRepository implements ports.CatalogChangesetRepository
// against "catalog_changeset": one document, its operations kept as the
// JSON they were queued as.
type CatalogChangesetRepository struct {
	Coll *mongo.Collection
}

var _ ports.CatalogChangesetRepository = (*CatalogChangesetRepository)(nil)

type changeDoc struct {
	ID bson.ObjectID `bson:"id"`
	Op string        `bson:"op"`
	By bson.ObjectID `bson:"by"`
	At time.Time     `bson:"at"`
}

type changesetDoc struct {
	ID        string      `bson:"_id"`
	Rev       int         `bson:"rev"`
	Changes   []changeDoc `bson:"changes"`
	UpdatedAt time.Time   `bson:"updatedAt"`
}

// Get implements ports.CatalogChangesetRepository.
func (r *CatalogChangesetRepository) Get(ctx context.Context) (*domain.CatalogChangeset, error) {
	var d changesetDoc
	err := r.Coll.FindOne(ctx, bson.M{"_id": changesetID}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return &domain.CatalogChangeset{}, nil
	}
	if err != nil {
		return nil, err
	}
	cs := &domain.CatalogChangeset{Rev: d.Rev, UpdatedAt: d.UpdatedAt}
	for _, c := range d.Changes {
		var op domain.CatalogOp
		if err := json.Unmarshal([]byte(c.Op), &op); err != nil {
			return nil, err
		}
		cs.Changes = append(cs.Changes, domain.CatalogChange{ID: c.ID, Op: op, By: c.By, At: c.At})
	}
	return cs, nil
}

// Save implements ports.CatalogChangesetRepository: a compare-and-set on
// rev, so two admins' writes never interleave unseen.
func (r *CatalogChangesetRepository) Save(ctx context.Context, cs *domain.CatalogChangeset, rev int) error {
	changes := make([]changeDoc, 0, len(cs.Changes))
	for _, c := range cs.Changes {
		op, err := json.Marshal(c.Op)
		if err != nil {
			return err
		}
		changes = append(changes, changeDoc{ID: c.ID, Op: string(op), By: c.By, At: c.At})
	}
	filter := bson.M{"_id": changesetID, "rev": rev}
	if rev == 0 {
		// The first write: there is no document yet, or one at rev 0.
		filter = bson.M{"_id": changesetID, "$or": bson.A{bson.M{"rev": 0}, bson.M{"rev": bson.M{"$exists": false}}}}
	}
	_, err := r.Coll.UpdateOne(ctx, filter,
		bson.M{"$set": bson.M{"rev": rev + 1, "changes": changes, "updatedAt": cs.UpdatedAt}},
		options.UpdateOne().SetUpsert(true))
	if mongo.IsDuplicateKeyError(err) {
		// The filter missed (another write came first) and the upsert hit
		// the existing _id.
		return domain.Coded("catalog.changeset_stale", errors.Join(err, domain.ErrConflict))
	}
	return err
}

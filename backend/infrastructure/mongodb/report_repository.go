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

// reportDoc mirrors domain.Report for BSON persistence.
type reportDoc struct {
	ID         bson.ObjectID           `bson:"_id,omitempty"`
	Reference  string                  `bson:"reference,omitempty"`
	Note       string                  `bson:"note,omitempty"`
	ReporterID bson.ObjectID           `bson:"reporterId"`
	TargetID   bson.ObjectID           `bson:"targetId"`
	TargetType domain.ReportTargetType `bson:"targetType"`
	Reason     string                  `bson:"reason"`
	Status     domain.ReportStatus     `bson:"status"`
	CreatedAt  time.Time               `bson:"createdAt"`
}

func fromDomainReport(r *domain.Report) reportDoc {
	return reportDoc{
		ID: r.ID, Reference: r.Reference, Note: r.Note, ReporterID: r.ReporterID, TargetID: r.TargetID,
		TargetType: r.TargetType, Reason: r.Reason, Status: r.Status, CreatedAt: r.CreatedAt,
	}
}

func (d reportDoc) toDomain() *domain.Report {
	return &domain.Report{
		ID: d.ID, Reference: d.Reference, Note: d.Note, ReporterID: d.ReporterID, TargetID: d.TargetID,
		TargetType: d.TargetType, Reason: d.Reason, Status: d.Status, CreatedAt: d.CreatedAt,
	}
}

// ReportRepository implements ports.ReportRepository against the
// "reports" collection.
type ReportRepository struct {
	Coll *mongo.Collection
	// Sequencer, when set, numbers each new report ("RPT-0331").
	Sequencer ports.Sequencer
}

var _ ports.ReportRepository = (*ReportRepository)(nil)

// Insert implements ports.ReportRepository.
func (r *ReportRepository) Insert(ctx context.Context, rp *domain.Report) error {
	if rp.ID.IsZero() {
		rp.ID = bson.NewObjectID()
	}
	if rp.Reference == "" && r.Sequencer != nil {
		n, err := r.Sequencer.Next(ctx, domain.ReportRefPrefix)
		if err != nil {
			return err
		}
		rp.Reference = domain.FormatReference(domain.ReportRefPrefix, n)
	}
	_, err := r.Coll.InsertOne(ctx, fromDomainReport(rp))
	return err
}

// FindByID implements ports.ReportRepository.
func (r *ReportRepository) FindByID(ctx context.Context, id bson.ObjectID) (*domain.Report, error) {
	var doc reportDoc
	err := r.Coll.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return doc.toDomain(), nil
}

// CountRecentByTarget implements ports.ReportRepository.
func (r *ReportRepository) CountRecentByTarget(ctx context.Context, targetID bson.ObjectID, since time.Time) (int, error) {
	n, err := r.Coll.CountDocuments(ctx, bson.M{"targetId": targetID, "createdAt": bson.M{"$gte": since}})
	return int(n), err
}

// ListByTarget implements ports.ReportRepository.
func (r *ReportRepository) ListByTarget(ctx context.Context, targetID bson.ObjectID) ([]*domain.Report, error) {
	return r.find(ctx, bson.M{"targetId": targetID})
}

// List implements ports.ReportRepository.
func (r *ReportRepository) List(ctx context.Context, status domain.ReportStatus) ([]*domain.Report, error) {
	filter := bson.M{}
	if status != "" {
		filter["status"] = status
	}
	return r.find(ctx, filter)
}

func (r *ReportRepository) find(ctx context.Context, filter bson.M) ([]*domain.Report, error) {
	cur, err := r.Coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []*domain.Report
	for cur.Next(ctx) {
		var doc reportDoc
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		out = append(out, doc.toDomain())
	}
	return out, cur.Err()
}

// Update implements ports.ReportRepository.
func (r *ReportRepository) Update(ctx context.Context, rp *domain.Report) error {
	res, err := r.Coll.ReplaceOne(ctx, bson.M{"_id": rp.ID}, fromDomainReport(rp))
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

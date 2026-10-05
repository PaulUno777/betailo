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

// reviewDoc mirrors domain.Review for BSON persistence.
type reviewCriteriaDoc struct {
	Health        int `bson:"health,omitempty"`
	Conformity    int `bson:"conformity,omitempty"`
	Communication int `bson:"communication,omitempty"`
	Timeliness    int `bson:"timeliness,omitempty"`
}

type reviewReplyDoc struct {
	Text string    `bson:"text"`
	At   time.Time `bson:"at"`
}

type reviewDoc struct {
	ID            bson.ObjectID     `bson:"_id,omitempty"`
	ReservationID bson.ObjectID     `bson:"reservationId"`
	AuthorID      bson.ObjectID     `bson:"authorId"`
	FarmID        bson.ObjectID     `bson:"farmId"`
	Rating        int               `bson:"rating"`
	Comment       string            `bson:"comment,omitempty"`
	Criteria      reviewCriteriaDoc `bson:"criteria,omitempty"`
	Tags          []string          `bson:"tags,omitempty"`
	Photos        []string          `bson:"photos,omitempty"`
	HelpfulCount  int               `bson:"helpfulCount"`
	HelpfulBy     []bson.ObjectID   `bson:"helpfulBy,omitempty"`
	Reply         *reviewReplyDoc   `bson:"reply,omitempty"`
	CreatedAt     time.Time         `bson:"createdAt,omitempty"`
}

func fromDomainReview(r *domain.Review) reviewDoc {
	d := reviewDoc{
		ID: r.ID, ReservationID: r.ReservationID, AuthorID: r.AuthorID,
		FarmID: r.FarmID, Rating: r.Rating, Comment: r.Comment,
		Criteria: reviewCriteriaDoc{Health: r.Criteria.Health, Conformity: r.Criteria.Conformity,
			Communication: r.Criteria.Communication, Timeliness: r.Criteria.Timeliness},
		Tags: r.Tags, Photos: r.Photos, HelpfulCount: r.HelpfulCount, CreatedAt: r.CreatedAt,
	}
	if r.Reply != nil {
		d.Reply = &reviewReplyDoc{Text: r.Reply.Text, At: r.Reply.At}
	}
	return d
}

func (d reviewDoc) toDomain() *domain.Review {
	r := &domain.Review{
		ID: d.ID, ReservationID: d.ReservationID, AuthorID: d.AuthorID,
		FarmID: d.FarmID, Rating: d.Rating, Comment: d.Comment,
		Criteria: domain.ReviewCriteria{Health: d.Criteria.Health, Conformity: d.Criteria.Conformity,
			Communication: d.Criteria.Communication, Timeliness: d.Criteria.Timeliness},
		Tags: d.Tags, Photos: d.Photos, HelpfulCount: d.HelpfulCount, CreatedAt: d.CreatedAt,
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = d.ID.Timestamp() // reviews written before createdAt existed
	}
	if d.Reply != nil {
		r.Reply = &domain.ReviewReply{Text: d.Reply.Text, At: d.Reply.At}
	}
	return r
}

// ReviewRepository implements ports.ReviewRepository against the
// "reviews" collection.
type ReviewRepository struct {
	Coll *mongo.Collection
}

var (
	_ ports.ReviewRepository = (*ReviewRepository)(nil)
	_ ports.ReviewStore      = (*ReviewRepository)(nil)
)

// Insert implements ports.ReviewRepository. The unique index on
// reservationId (see infrastructure/mongodb.Bootstrap) is what actually
// enforces "one review per reservation" — a race surfaces here as
// domain.ErrConflict rather than the usecase needing a separate
// check-then-insert.
func (r *ReviewRepository) Insert(ctx context.Context, rv *domain.Review) error {
	if rv.ID.IsZero() {
		rv.ID = bson.NewObjectID()
	}
	_, err := r.Coll.InsertOne(ctx, fromDomainReview(rv))
	if err == nil {
		return nil
	}
	var writeErr mongo.WriteException
	if errors.As(err, &writeErr) && writeErr.HasErrorCode(duplicateKeyCode) {
		return domain.ErrConflict
	}
	return err
}

// ListByFarm implements ports.ReviewRepository.
func (r *ReviewRepository) ListByFarm(ctx context.Context, farmID bson.ObjectID) ([]*domain.Review, error) {
	cur, err := r.Coll.Find(ctx, bson.M{"farmId": farmID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []*domain.Review
	for cur.Next(ctx) {
		var doc reviewDoc
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		out = append(out, doc.toDomain())
	}
	return out, cur.Err()
}

// Summary implements ports.ReviewRepository via one aggregation: a
// single $group computes the average, count and 1-5 histogram in one
// round trip rather than pulling every review document into Go to
// average client-side.
func (r *ReviewRepository) Summary(ctx context.Context, farmID bson.ObjectID) (ports.ReviewSummary, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"farmId": farmID}}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$rating",
			"count": bson.M{"$sum": 1},
		}}},
	}

	cur, err := r.Coll.Aggregate(ctx, pipeline)
	if err != nil {
		return ports.ReviewSummary{}, err
	}
	defer cur.Close(ctx)

	summary := ports.ReviewSummary{Histogram: map[int]int{}}
	criteria, err := r.criteriaAverages(ctx, farmID)
	if err != nil {
		return ports.ReviewSummary{}, err
	}
	summary.Criteria = criteria
	var weightedSum, total float64
	for cur.Next(ctx) {
		var row struct {
			Rating int `bson:"_id"`
			Count  int `bson:"count"`
		}
		if err := cur.Decode(&row); err != nil {
			return ports.ReviewSummary{}, err
		}
		summary.Histogram[row.Rating] = row.Count
		summary.Count += row.Count
		weightedSum += float64(row.Rating) * float64(row.Count)
		total += float64(row.Count)
	}
	if err := cur.Err(); err != nil {
		return ports.ReviewSummary{}, err
	}
	if total > 0 {
		summary.Average = weightedSum / total
	}
	return summary, nil
}

// criteriaAverages averages each detailed score over the reviews that
// gave one ($avg skips the nulls standing in for "not rated").
func (r *ReviewRepository) criteriaAverages(ctx context.Context, farmID bson.ObjectID) (ports.CriteriaAverages, error) {
	rated := func(field string) bson.M {
		return bson.M{"$avg": bson.M{"$cond": bson.A{bson.M{"$gt": bson.A{"$criteria." + field, 0}}, "$criteria." + field, nil}}}
	}
	cur, err := r.Coll.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"farmId": farmID}}},
		{{Key: "$group", Value: bson.M{
			"_id": nil, "health": rated("health"), "conformity": rated("conformity"),
			"communication": rated("communication"), "timeliness": rated("timeliness"),
		}}},
	})
	if err != nil {
		return ports.CriteriaAverages{}, err
	}
	var rows []struct {
		Health, Conformity, Communication, Timeliness *float64
	}
	if err := cur.All(ctx, &rows); err != nil {
		return ports.CriteriaAverages{}, err
	}
	if len(rows) == 0 {
		return ports.CriteriaAverages{}, nil
	}
	val := func(p *float64) float64 {
		if p == nil {
			return 0
		}
		return *p
	}
	return ports.CriteriaAverages{
		Health: val(rows[0].Health), Conformity: val(rows[0].Conformity),
		Communication: val(rows[0].Communication), Timeliness: val(rows[0].Timeliness),
	}, nil
}

func reviewFilterQuery(farmID bson.ObjectID, f ports.ReviewFilter) bson.M {
	q := bson.M{"farmId": farmID}
	switch f {
	case ports.ReviewFilterPhotos:
		q["photos.0"] = bson.M{"$exists": true}
	case ports.ReviewFilterFive:
		q["rating"] = 5
	case ports.ReviewFilterCritical:
		q["rating"] = bson.M{"$lte": 3}
	}
	return q
}

// ListPage implements ports.ReviewStore.
func (r *ReviewRepository) ListPage(ctx context.Context, farmID bson.ObjectID, f ports.ReviewFilter, page, pageSize int) ([]*domain.Review, int, error) {
	q := reviewFilterQuery(farmID, f)
	total, err := r.Coll.CountDocuments(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	cur, err := r.Coll.Find(ctx, q, options.Find().
		SetSort(bson.D{{Key: "_id", Value: -1}}).
		SetSkip(int64((page-1)*pageSize)).SetLimit(int64(pageSize)).
		SetProjection(bson.M{"helpfulBy": 0}))
	if err != nil {
		return nil, 0, err
	}
	var docs []reviewDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, 0, err
	}
	out := make([]*domain.Review, 0, len(docs))
	for _, d := range docs {
		out = append(out, d.toDomain())
	}
	return out, int(total), nil
}

// FindByID implements ports.ReviewStore.
func (r *ReviewRepository) FindByID(ctx context.Context, id bson.ObjectID) (*domain.Review, error) {
	var d reviewDoc
	err := r.Coll.FindOne(ctx, bson.M{"_id": id}, options.FindOne().SetProjection(bson.M{"helpfulBy": 0})).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return d.toDomain(), nil
}

// SetReply implements ports.ReviewStore.
func (r *ReviewRepository) SetReply(ctx context.Context, id bson.ObjectID, reply domain.ReviewReply) error {
	res, err := r.Coll.UpdateOne(ctx,
		bson.M{"_id": id, "reply": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"reply": reviewReplyDoc{Text: reply.Text, At: reply.At}}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		if n, err := r.Coll.CountDocuments(ctx, bson.M{"_id": id}); err != nil {
			return err
		} else if n == 0 {
			return domain.ErrNotFound
		}
		return domain.ErrConflict
	}
	return nil
}

// AddHelpfulVote implements ports.ReviewStore: one conditional update
// both records the voter and bumps the count, so it can't double count.
func (r *ReviewRepository) AddHelpfulVote(ctx context.Context, id, userID bson.ObjectID) (bool, error) {
	res, err := r.Coll.UpdateOne(ctx,
		bson.M{"_id": id, "helpfulBy": bson.M{"$ne": userID}},
		bson.M{"$push": bson.M{"helpfulBy": userID}, "$inc": bson.M{"helpfulCount": 1}})
	if err != nil {
		return false, err
	}
	if res.MatchedCount == 0 {
		if n, err := r.Coll.CountDocuments(ctx, bson.M{"_id": id}); err != nil {
			return false, err
		} else if n == 0 {
			return false, domain.ErrNotFound
		}
		return false, nil
	}
	return true, nil
}

// VotedBy implements ports.ReviewStore.
func (r *ReviewRepository) VotedBy(ctx context.Context, userID bson.ObjectID, ids []bson.ObjectID) (map[bson.ObjectID]bool, error) {
	out := map[bson.ObjectID]bool{}
	if len(ids) == 0 || userID.IsZero() {
		return out, nil
	}
	cur, err := r.Coll.Find(ctx, bson.M{"_id": bson.M{"$in": ids}, "helpfulBy": userID}, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return nil, err
	}
	var docs []struct {
		ID bson.ObjectID `bson:"_id"`
	}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	for _, d := range docs {
		out[d.ID] = true
	}
	return out, nil
}

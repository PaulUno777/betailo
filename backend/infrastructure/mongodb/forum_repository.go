package mongodb

import (
	"context"
	"errors"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// forumReplyDoc mirrors domain.ForumReply for BSON persistence.
type forumReplyDoc struct {
	AuthorID  bson.ObjectID `bson:"authorId"`
	Body      string        `bson:"body"`
	CreatedAt time.Time     `bson:"createdAt"`
}

// forumPostDoc mirrors domain.ForumPost for BSON persistence.
type forumPostDoc struct {
	ID             bson.ObjectID   `bson:"_id,omitempty"`
	AuthorID       bson.ObjectID   `bson:"authorId"`
	Category       string          `bson:"category"`
	Title          string          `bson:"title"`
	Body           string          `bson:"body"`
	ListingID      *bson.ObjectID  `bson:"listingId,omitempty"`
	CreatedAt      time.Time       `bson:"createdAt"`
	Replies        []forumReplyDoc `bson:"replies,omitempty"`
	BestReplyIndex *int            `bson:"bestReplyIndex,omitempty"`
}

func fromDomainForumPost(p *domain.ForumPost) forumPostDoc {
	doc := forumPostDoc{
		ID: p.ID, AuthorID: p.AuthorID, Category: p.Category, Title: p.Title,
		Body: p.Body, ListingID: p.ListingID, CreatedAt: p.CreatedAt, BestReplyIndex: p.BestReplyIndex,
	}
	for _, rep := range p.Replies {
		doc.Replies = append(doc.Replies, forumReplyDoc{AuthorID: rep.AuthorID, Body: rep.Body, CreatedAt: rep.CreatedAt})
	}
	return doc
}

func (d forumPostDoc) toDomain() *domain.ForumPost {
	p := &domain.ForumPost{
		ID: d.ID, AuthorID: d.AuthorID, Category: d.Category, Title: d.Title,
		Body: d.Body, ListingID: d.ListingID, CreatedAt: d.CreatedAt, BestReplyIndex: d.BestReplyIndex,
	}
	for _, rep := range d.Replies {
		p.Replies = append(p.Replies, domain.ForumReply{AuthorID: rep.AuthorID, Body: rep.Body, CreatedAt: rep.CreatedAt})
	}
	return p
}

// ForumRepository implements ports.ForumRepository against the
// "forum_posts" collection.
type ForumRepository struct {
	Coll *mongo.Collection
}

var _ ports.ForumRepository = (*ForumRepository)(nil)

// Insert implements ports.ForumRepository.
func (r *ForumRepository) Insert(ctx context.Context, p *domain.ForumPost) error {
	if p.ID.IsZero() {
		p.ID = bson.NewObjectID()
	}
	_, err := r.Coll.InsertOne(ctx, fromDomainForumPost(p))
	return err
}

// FindByID implements ports.ForumRepository.
func (r *ForumRepository) FindByID(ctx context.Context, id bson.ObjectID) (*domain.ForumPost, error) {
	var doc forumPostDoc
	err := r.Coll.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return doc.toDomain(), nil
}

// List implements ports.ForumRepository. "popular"/"unanswered" sort by
// reply count via $expr/$size rather than an extra denormalized
// counter — the replies array is bounded (see the architecture
// document), so this stays a cheap in-place sort, not a full scan.
func (r *ForumRepository) List(ctx context.Context, filter ports.ForumListFilter) ([]*domain.ForumPost, error) {
	match := bson.M{}
	if filter.Category != "" {
		match["category"] = filter.Category
	}
	if filter.Query != "" {
		match["$or"] = bson.A{
			bson.M{"title": bson.M{"$regex": regexp.QuoteMeta(filter.Query), "$options": "i"}},
			bson.M{"body": bson.M{"$regex": regexp.QuoteMeta(filter.Query), "$options": "i"}},
		}
	}
	if filter.Sort == "unanswered" {
		match["$expr"] = bson.M{"$eq": bson.A{bson.M{"$size": bson.M{"$ifNull": bson.A{"$replies", bson.A{}}}}, 0}}
	}

	pipeline := mongo.Pipeline{{{Key: "$match", Value: match}}}
	if filter.Sort == "popular" {
		pipeline = append(pipeline,
			bson.D{{Key: "$addFields", Value: bson.M{"replyCount": bson.M{"$size": bson.M{"$ifNull": bson.A{"$replies", bson.A{}}}}}}},
			bson.D{{Key: "$sort", Value: bson.D{{Key: "replyCount", Value: -1}, {Key: "createdAt", Value: -1}}}},
		)
	} else {
		pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{{Key: "createdAt", Value: -1}}}})
	}

	cur, err := r.Coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []*domain.ForumPost
	for cur.Next(ctx) {
		var doc forumPostDoc
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		out = append(out, doc.toDomain())
	}
	return out, cur.Err()
}

// AddReply implements ports.ForumRepository.
func (r *ForumRepository) AddReply(ctx context.Context, postID bson.ObjectID, reply domain.ForumReply) error {
	res, err := r.Coll.UpdateOne(ctx, bson.M{"_id": postID}, bson.M{"$push": bson.M{
		"replies": forumReplyDoc{AuthorID: reply.AuthorID, Body: reply.Body, CreatedAt: reply.CreatedAt},
	}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// SetBestReplyIndex implements ports.ForumRepository.
func (r *ForumRepository) SetBestReplyIndex(ctx context.Context, postID bson.ObjectID, index int) error {
	res, err := r.Coll.UpdateOne(ctx, bson.M{"_id": postID}, bson.M{"$set": bson.M{"bestReplyIndex": index}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return domain.ErrNotFound
	}
	return nil
}

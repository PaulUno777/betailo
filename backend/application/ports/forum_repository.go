package ports

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

// ForumListFilter narrows/orders a forum listing — every field is
// optional (zero value = no filter/default order).
type ForumListFilter struct {
	Category string
	Query    string
	// Sort is "recent" (default), "popular" (most replies), or
	// "unanswered" (zero replies).
	Sort string
}

// ForumRepository persists forum posts and their embedded replies —
// bounded, always read/written with their parent post, per the
// architecture document's data-model rationale for this collection.
type ForumRepository interface {
	Insert(ctx context.Context, p *domain.ForumPost) error

	// FindByID returns domain.ErrNotFound if no post has this ID.
	FindByID(ctx context.Context, id bson.ObjectID) (*domain.ForumPost, error)

	List(ctx context.Context, filter ForumListFilter) ([]*domain.ForumPost, error)

	// AddReply appends to the post's embedded replies array.
	AddReply(ctx context.Context, postID bson.ObjectID, reply domain.ForumReply) error

	// SetBestReplyIndex records which reply the post's author marked as
	// the answer.
	SetBestReplyIndex(ctx context.Context, postID bson.ObjectID, index int) error
}

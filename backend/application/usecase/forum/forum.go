// Package forum implements the community layer: posts, replies and
// categories, deliberately kept simple per the architecture document —
// it shares Trust's reporting mechanism rather than having its own
// moderation.
package forum

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// CreatePost is the usecase behind "Nouvelle discussion".
type CreatePost struct {
	Forum ports.ForumRepository
	Clock ports.Clock
}

type CreatePostInput struct {
	AuthorID  bson.ObjectID
	Category  string
	Title     string
	Body      string
	ListingID *bson.ObjectID
}

func (uc *CreatePost) Execute(ctx context.Context, in CreatePostInput) (*domain.ForumPost, error) {
	if in.Category == "" || in.Title == "" || in.Body == "" {
		return nil, fmt.Errorf("forum: category, title and body are required: %w", domain.ErrInvalid)
	}
	p := &domain.ForumPost{
		ID: bson.NewObjectID(), AuthorID: in.AuthorID, Category: in.Category,
		Title: in.Title, Body: in.Body, ListingID: in.ListingID, CreatedAt: uc.Clock.Now(),
	}
	if err := uc.Forum.Insert(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// ListPosts backs the forum's index (category filter, search, and the
// recent/popular/unanswered tabs the design shows).
type ListPosts struct {
	Forum ports.ForumRepository
}

func (uc *ListPosts) Execute(ctx context.Context, filter ports.ForumListFilter) ([]*domain.ForumPost, error) {
	return uc.Forum.List(ctx, filter)
}

// GetPost backs a single thread view.
type GetPost struct {
	Forum ports.ForumRepository
}

func (uc *GetPost) Execute(ctx context.Context, id bson.ObjectID) (*domain.ForumPost, error) {
	return uc.Forum.FindByID(ctx, id)
}

// AddReply appends a reply to an existing post.
type AddReply struct {
	Forum ports.ForumRepository
	Clock ports.Clock
}

type AddReplyInput struct {
	PostID   bson.ObjectID
	AuthorID bson.ObjectID
	Body     string
}

func (uc *AddReply) Execute(ctx context.Context, in AddReplyInput) error {
	if in.Body == "" {
		return fmt.Errorf("forum: reply body is required: %w", domain.ErrInvalid)
	}
	if _, err := uc.Forum.FindByID(ctx, in.PostID); err != nil {
		return err
	}
	return uc.Forum.AddReply(ctx, in.PostID, domain.ForumReply{
		AuthorID: in.AuthorID, Body: in.Body, CreatedAt: uc.Clock.Now(),
	})
}

// MarkBestAnswer lets a post's author flag one reply as the answer that
// resolved their question.
type MarkBestAnswer struct {
	Forum ports.ForumRepository
}

type MarkBestAnswerInput struct {
	PostID     bson.ObjectID
	AuthorID   bson.ObjectID
	ReplyIndex int
}

// Execute returns domain.ErrForbidden if the caller didn't author the
// post, and domain.ErrInvalid if the reply index is out of range.
func (uc *MarkBestAnswer) Execute(ctx context.Context, in MarkBestAnswerInput) error {
	p, err := uc.Forum.FindByID(ctx, in.PostID)
	if err != nil {
		return err
	}
	if p.AuthorID != in.AuthorID {
		return fmt.Errorf("forum: only the post's author can mark a best answer: %w", domain.ErrForbidden)
	}
	if in.ReplyIndex < 0 || in.ReplyIndex >= len(p.Replies) {
		return fmt.Errorf("forum: reply index out of range: %w", domain.ErrInvalid)
	}
	return uc.Forum.SetBestReplyIndex(ctx, in.PostID, in.ReplyIndex)
}

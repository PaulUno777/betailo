// Package forum implements the community layer: posts, replies and
// categories, deliberately kept simple per the architecture document —
// it shares Trust's reporting mechanism rather than having its own
// moderation.
package forum

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/notify"
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
		return nil, domain.Coded("forum.post_required", fmt.Errorf("forum: category, title and body are required: %w", domain.ErrInvalid))
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
	// Notifications tells the post's author about the reply. Optional.
	Notifications ports.NotificationRepository
}

// maxReplyLen bounds a reply's body.
const maxReplyLen = 2000

type AddReplyInput struct {
	PostID   bson.ObjectID
	AuthorID bson.ObjectID
	Body     string
}

func (uc *AddReply) Execute(ctx context.Context, in AddReplyInput) error {
	body := strings.TrimSpace(in.Body)
	if body == "" {
		return domain.Coded("forum.reply_required", fmt.Errorf("forum: reply body is required: %w", domain.ErrInvalid))
	}
	if utf8.RuneCountInString(body) > maxReplyLen {
		return fmt.Errorf("forum: reply must be at most %d characters: %w", maxReplyLen, domain.ErrInvalid)
	}
	post, err := uc.Forum.FindByID(ctx, in.PostID)
	if err != nil {
		return err
	}
	now := uc.Clock.Now()
	if err := uc.Forum.AddReply(ctx, in.PostID, domain.ForumReply{AuthorID: in.AuthorID, Body: body, CreatedAt: now}); err != nil {
		return err
	}
	if post.AuthorID != in.AuthorID {
		notify.Send(ctx, uc.Notifications, now, notify.Notice{
			UserID: post.AuthorID, Type: domain.NotifForumReply, Params: map[string]string{"title": post.Title},
			Link: "/forum/" + post.ID.Hex(), Fallback: "Nouvelle réponse à « " + post.Title + " ».",
		})
	}
	return nil
}

// MarkBestAnswer lets a post's author flag one reply as the answer that
// resolved their question.
type MarkBestAnswer struct {
	Forum ports.ForumRepository
	Clock ports.Clock
	// Notifications tells the reply's author. Optional.
	Notifications ports.NotificationRepository
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
	if err := uc.Forum.SetBestReplyIndex(ctx, in.PostID, in.ReplyIndex); err != nil {
		return err
	}
	if replier := p.Replies[in.ReplyIndex].AuthorID; replier != p.AuthorID && uc.Clock != nil {
		notify.Send(ctx, uc.Notifications, uc.Clock.Now(), notify.Notice{
			UserID: replier, Type: domain.NotifBestAnswer, Params: map[string]string{"title": p.Title},
			Link: "/forum/" + p.ID.Hex(), Fallback: "Votre réponse à « " + p.Title + " » a été choisie comme meilleure réponse.",
		})
	}
	return nil
}

package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ForumReply is one entry in a ForumPost's embedded replies list —
// always read and written together with its parent post, and bounded
// in size (no unlimited thread), the case MongoDB's document embedding
// targets.
type ForumReply struct {
	AuthorID  bson.ObjectID
	Body      string
	CreatedAt time.Time
}

// ForumPost handles community categories, posts and replies,
// optionally linked to a listing. Kept deliberately simple in the MVP:
// it builds trust and content around the marketplace rather than being
// a general-purpose forum, and shares Trust's reporting mechanism
// rather than having its own moderation — see the "Forum" section of
// the architecture document. No repository lives for it yet — see
// domain/user.go's comment.
type ForumPost struct {
	ID        bson.ObjectID
	AuthorID  bson.ObjectID
	Category  string
	Title     string
	Body      string
	ListingID *bson.ObjectID
	CreatedAt time.Time
	Replies   []ForumReply

	// BestReplyIndex, when set, points into Replies — the post's author
	// marking one reply as the answer that resolved their question.
	BestReplyIndex *int
}

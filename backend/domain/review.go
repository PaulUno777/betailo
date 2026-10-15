package domain

import (
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Review holds ratings/comments left after a confirmed reservation —
// always verified (only reservable, never freely posted), per the
// "Trust & anti-scam" section of the architecture document. No
// repository lives for it yet — see domain/user.go's comment.
type Review struct {
	ID            bson.ObjectID
	ReservationID bson.ObjectID
	AuthorID      bson.ObjectID

	// FarmID is denormalized at creation time (resolved once via
	// reservation -> listing -> cycle -> farm) so RatingSummary and the
	// marketplace read model can query reviews by farm directly, instead
	// of joining four collections on every read.
	FarmID  bson.ObjectID
	Rating  int
	Comment string

	// Criteria are the optional detailed scores ("En détail").
	Criteria ReviewCriteria
	// Tags are picked from the Catalog's review tags ("Ce qui vous a plu").
	Tags []string
	// Photos are object-storage keys, like listing photos.
	Photos []string
	// HelpfulCount is how many other users marked it "Utile".
	HelpfulCount int
	// Reply is the farmer's one public answer, if any.
	Reply     *ReviewReply
	CreatedAt time.Time
}

// ReviewCriteria scores each aspect from 1 to 5; 0 means not rated.
type ReviewCriteria struct {
	Health        int // santé des animaux
	Conformity    int // conforme à l'annonce
	Communication int
	Timeliness    int // respect des délais
}

// Validate checks every score is 0 (not rated) or 1-5.
func (c ReviewCriteria) Validate() error {
	for _, v := range []int{c.Health, c.Conformity, c.Communication, c.Timeliness} {
		if v < 0 || v > 5 {
			return fmt.Errorf("review: criteria scores must be between 1 and 5: %w", ErrInvalid)
		}
	}
	return nil
}

// ReviewReply is a farmer's public answer to a review.
type ReviewReply struct {
	Text string
	At   time.Time
}

// Review limits.
const (
	MaxReviewCommentLen = 1000
	MaxReviewReplyLen   = 500
	MaxReviewPhotos     = 4
)

// NormalizeReviewTags dedupes tags and rejects any outside known (the
// Catalog's active review tags).
func NormalizeReviewTags(tags, known []string) ([]string, error) {
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if !slices.Contains(known, t) {
			return nil, fmt.Errorf("review: unknown tag %q: %w", t, ErrInvalid)
		}
		if !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	return out, nil
}

// ValidateReviewText trims and bounds a comment or reply.
func ValidateReviewText(text string, max int) (string, error) {
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) > max {
		return "", fmt.Errorf("review: text must be at most %d characters: %w", max, ErrInvalid)
	}
	return text, nil
}

// PublicName is how a reviewer appears on a review: first name and the
// initial of the last ("Joël T."), never an email. Falls back to a
// neutral label when no display name is set.
func PublicName(displayName string) string {
	parts := strings.Fields(displayName)
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	default:
		last, _ := utf8.DecodeRuneInString(parts[len(parts)-1])
		return parts[0] + " " + string(last) + "."
	}
}

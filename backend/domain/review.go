package domain

import "go.mongodb.org/mongo-driver/v2/bson"

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
}

package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ReportTargetType is what a Report points at. Unlike the old
// relational schema, MongoDB enforces no foreign key here either way,
// so nothing is lost by keeping the same polymorphic
// targetId/targetType pair — see the "REPORTS" bullet in the "Modèle de
// données" section of the architecture document.
type ReportTargetType string

const (
	ReportTargetListing   ReportTargetType = "listing"
	ReportTargetProfile   ReportTargetType = "profile"
	ReportTargetForumPost ReportTargetType = "forum_post"
)

// ReportStatus is a Report's moderation state. UnderReview is set
// automatically once a target crosses the repeated-report threshold
// (see "3. Signalement anti-scam et reaction automatique").
type ReportStatus string

const (
	ReportOpen        ReportStatus = "ouvert"
	ReportUnderReview ReportStatus = "sous_revue"
	ReportHandled     ReportStatus = "traite"
	ReportRejected    ReportStatus = "rejete"
)

// Report computes the trust score and handles reports/flagging
// (anti-scam). It's not a secondary checkbox: profile verification,
// successful-transaction history, verified reviews, community
// reporting and simple detection rules are combined signals — see the
// "Trust & anti-scam" section of the architecture document. No
// repository lives for it yet — see domain/user.go's comment.
//
// TargetID points at whichever collection TargetType names — every
// target type happens to key its documents with an ObjectId, so the
// field stays concretely typed rather than falling back to `any`.
type Report struct {
	ID         bson.ObjectID
	ReporterID bson.ObjectID
	TargetID   bson.ObjectID
	TargetType ReportTargetType
	Reason     string
	Status     ReportStatus
	CreatedAt  time.Time
}

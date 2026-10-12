package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// NotificationStatus tracks whether internal/worker has processed a
// notification yet. There is no real email/SMS provider behind this —
// "sent" means the worker logged it — but the outbox shape (enqueue now,
// process asynchronously) is the same one a real channel would plug
// into later, per the architecture document's async-worker design.
type NotificationStatus string

const (
	NotificationPending NotificationStatus = "pending"
	NotificationSent    NotificationStatus = "sent"
)

// Notification is one outbox entry: enqueued synchronously by a usecase
// (an unpaid reservation expiring, a report crossing the moderation
// threshold), drained by internal/worker on its tick.
type Notification struct {
	ID        bson.ObjectID
	UserID    bson.ObjectID
	Type      string
	Message   string
	Status    NotificationStatus
	CreatedAt time.Time
	SentAt    *time.Time

	// Params carry the values the client interpolates into its own
	// localized template for Type ({"ref": "RSV-0042", "title": ...}).
	// Message stays as a French fallback for logs and unknown types.
	Params map[string]string
	// Link is the in-app route the notification opens.
	Link string
	// ReadAt is set once the user has seen it in the app.
	ReadAt *time.Time
}

// Notification types. The client keeps one localized template per type;
// Params fill it in.
const (
	NotifReservationCreated   = "reservation_created"   // to the farmer
	NotifReservationPaid      = "reservation_paid"      // to the farmer
	NotifReservationCancelled = "reservation_cancelled" // to the farmer
	NotifReservationExpired   = "reservation_expired"   // to the buyer
	NotifReservationDelivered = "reservation_delivered" // to the buyer
	NotifEscrowReleased       = "escrow_released"       // to the farmer
	NotifEscrowAutoReleased   = "escrow_auto_released"  // to the buyer
	NotifEscrowRefunded       = "escrow_refunded"       // to the buyer
	NotifDisputeOpened        = "dispute_opened"        // to the farmer
	NotifReviewReceived       = "review_received"       // to the farmer
	NotifReviewReply          = "review_reply"          // to the reviewer
	NotifCycleUpdate          = "cycle_update"          // to buyers of the cycle
	NotifForumReply           = "forum_reply"           // to the post's author
	NotifFarmApproved         = "farm_approved"         // to the farm's owner
	NotifFarmRejected         = "farm_rejected"         // to the farm's owner
	NotifFarmSuspended        = "farm_suspended"        // to the farm's owner
	NotifBestAnswer           = "best_answer"           // to the reply's author
)

// In-app routes notifications link to.
const (
	LinkBuyerReservations  = "/account/reservations"
	LinkFarmerReservations = "/portal/reservations"
	LinkSellerFarms        = "/portal/farms"
)

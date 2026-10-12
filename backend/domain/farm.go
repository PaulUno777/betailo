package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Farm manages a farmer's operation: location, species raised. No
// repository lives for it yet — see domain/user.go's comment.
type Farm struct {
	ID       bson.ObjectID
	OwnerID  bson.ObjectID
	Name     string
	Location string // the town, as the farmer writes it
	// Country is an ISO 3166-1 alpha-2 code (DefaultCountry when not
	// given) and Region its first-level division: one of KnownRegions
	// for countries that have a list, free text elsewhere. Together they
	// back the marketplace's region filter in any country.
	Country   string
	Region    string
	Species   []string
	CreatedAt time.Time

	// Description is the farmer's own pitch, shown on the public farm
	// profile. Optional.
	Description string
	// ContactPhone is E.164 (see NormalizePhone). Optional. It powers
	// the "Contacter l'éleveur" WhatsApp/call link, and the API only
	// shows it to signed-in users.
	ContactPhone string

	// Status is where the farm stands with the platform's review: only
	// an approved farm's listings reach the marketplace.
	Status FarmStatus
	// Verification is what the owner submitted for that review and
	// what staff decided.
	Verification FarmVerification
}

// FarmStatus is a farm's review state.
type FarmStatus string

const (
	// FarmDraft: created, proof not submitted yet.
	FarmDraft FarmStatus = "draft"
	// FarmPending: submitted, waiting for staff.
	FarmPending FarmStatus = "pending"
	// FarmApproved: may publish.
	FarmApproved FarmStatus = "approved"
	// FarmRejected: staff said no (Verification.RejectionReason says
	// why); the owner can fix and submit again.
	FarmRejected FarmStatus = "rejected"
	// FarmSuspended: staff took an approved farm off the marketplace.
	FarmSuspended FarmStatus = "suspended"
)

// CanSell reports whether the farm's listings may be on the marketplace.
func (s FarmStatus) CanSell() bool { return s == FarmApproved }

// CanSubmit reports whether the owner may (re)submit it for review.
func (s FarmStatus) CanSubmit() bool { return s == FarmDraft || s == FarmRejected }

// FarmVerification is the proof behind a farm's review. The document
// and photos are private: only the owner and staff get to see them.
type FarmVerification struct {
	// IDDocumentKey is the storage key of the owner's identity document.
	IDDocumentKey string
	// PhotoKeys are storage keys of photos of the farm itself.
	PhotoKeys   []string
	SubmittedAt *time.Time

	ReviewedAt *time.Time
	// ReviewedBy is the staff member who decided; zero when the
	// platform approved it on its own (AutoApproved).
	ReviewedBy bson.ObjectID
	// AutoApproved marks a farm approved by the FARM_AUTO_APPROVE test
	// switch rather than by a person, so staff can review it later.
	AutoApproved    bool
	RejectionReason string
}

// Farm verification limits.
const (
	MinFarmProofPhotos    = 1
	MaxFarmProofPhotos    = 3
	MaxRejectionReasonLen = 500
	// DefaultMaxFarmsPerAccount applies when MAX_FARMS_PER_ACCOUNT
	// isn't set.
	DefaultMaxFarmsPerAccount = 1
)

// FarmProofPrefix is the storage prefix a farm's proof files live
// under; keys outside it were not issued for this farm.
func FarmProofPrefix(farmID bson.ObjectID) string {
	return "farm-verifications/" + farmID.Hex() + "/"
}

// MaxFarmDescriptionLen bounds Farm.Description.
const MaxFarmDescriptionLen = 1000

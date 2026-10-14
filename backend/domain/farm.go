package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Farm manages a farmer's operation: location, species raised. No
// repository lives for it yet — see domain/user.go's comment.
type Farm struct {
	ID bson.ObjectID
	// Reference is the human code staff and owners quote ("FRM-0212"),
	// assigned from a counter at creation.
	Reference string
	OwnerID   bson.ObjectID
	Name      string
	Location  string // the town, as the farmer writes it
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
	// IDDocumentType says which document the owner sent.
	IDDocumentType IDDocumentType
	// IDDocumentKey is the storage key of the document's front;
	// IDDocumentBackKey its back, for documents that have one (see
	// IDDocumentType.HasBack).
	IDDocumentKey     string
	IDDocumentBackKey string
	// PhotoKeys are storage keys of photos of the farm itself.
	PhotoKeys   []string
	SubmittedAt *time.Time
	// Submissions counts how many times the farm was sent for review:
	// above 1, staff are looking at a farm they already refused once.
	Submissions int

	ReviewedAt *time.Time
	// ReviewedBy is the staff member who decided; zero when the
	// platform approved it on its own (AutoApproved).
	ReviewedBy bson.ObjectID
	// AutoApproved marks a farm approved by the FARM_AUTO_APPROVE test
	// switch rather than by a person, so staff can review it later.
	AutoApproved bool
	// RejectionCode is the kind of refusal staff picked;
	// RejectionReason the message they wrote to the owner.
	RejectionCode   RejectionCode
	RejectionReason string
}

// IDDocumentType is a kind of identity document the platform accepts.
type IDDocumentType string

const (
	IDNationalCard   IDDocumentType = "national_id"
	IDPassport       IDDocumentType = "passport"
	IDDrivingLicence IDDocumentType = "driving_licence"
)

// IsValid reports whether t is an accepted document.
func (t IDDocumentType) IsValid() bool {
	return t == IDNationalCard || t == IDPassport || t == IDDrivingLicence
}

// HasBack reports whether the document has a second side to send: a
// passport's identity page stands alone, a card has two faces.
func (t IDDocumentType) HasBack() bool { return t == IDNationalCard || t == IDDrivingLicence }

// RejectionCode is why staff refused a farm, picked from a short list
// so refusals read the same from one reviewer to the next.
type RejectionCode string

const (
	RejectIDUnreadable    RejectionCode = "id_unreadable"
	RejectNameMismatch    RejectionCode = "name_mismatch"
	RejectPhotosNotRecent RejectionCode = "photos_not_recent"
	RejectFarmNotFound    RejectionCode = "farm_not_found"
	RejectOther           RejectionCode = "other"
)

// RejectionCodes lists the refusal reasons in the order staff see them.
var RejectionCodes = []RejectionCode{RejectIDUnreadable, RejectNameMismatch, RejectPhotosNotRecent, RejectFarmNotFound, RejectOther}

// IsValid reports whether c is a known refusal reason.
func (c RejectionCode) IsValid() bool {
	for _, known := range RejectionCodes {
		if c == known {
			return true
		}
	}
	return false
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

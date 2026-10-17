package domain

import (
	"fmt"
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
	// given) and Region its first-level division: one of the Catalog's regions
	// for countries that have a list, free text elsewhere. Together they
	// back the marketplace's region filter in any country.
	Country   string
	Region    string
	Species   []string
	CreatedAt time.Time

	// Description is the farmer's own pitch, shown on the public farm
	// profile. Optional.
	Description string
	// Contacts are the numbers buyers reach the farm on, in the order
	// the owner listed them (at most the platform's contact limit, see
	// DefaultMaxFarmContacts). Optional. They power the "Contacter
	// l'éleveur" links, and the API only shows them to signed-in users.
	Contacts []FarmContact
	// LogoKey and BannerKey are the storage keys of the farm's logo and
	// banner (see FarmBrandingPrefix); empty when the owner set none,
	// and the app then shows its default artwork.
	LogoKey   string
	BannerKey string

	// Status is where the farm stands with the platform's review: only
	// an approved farm's listings reach the marketplace.
	Status FarmStatus
	// Verification is what the owner submitted for that review and
	// what staff decided.
	Verification FarmVerification
}

// CheckRaises refuses a species the farm didn't declare: staff review a
// farm for the activity it presented, so its cycles and listings stay
// within it. The owner adds a species to the farm first.
func (f *Farm) CheckRaises(species string) error {
	for _, s := range f.Species {
		if s == species {
			return nil
		}
	}
	return CodedWith("farm.species_not_declared", map[string]any{"species": species}, fmt.Errorf("farm %s doesn't raise %q: %w", f.ID.Hex(), species, ErrInvalid))
}

// FarmContact is one phone number of a farm, in E.164 (see
// NormalizePhone). WhatsApp says whether buyers can also write to it
// on WhatsApp; otherwise it is a number to call only.
type FarmContact struct {
	Phone    string
	WhatsApp bool
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
	// ReviewedByRole is that staff member's role at the time, which the
	// owner is told ("refusée par l'équipe Modération").
	ReviewedByRole StaffRole
	// AutoApproved marks a farm approved by the FARM_AUTO_APPROVE test
	// switch rather than by a person, so staff can review it later.
	AutoApproved bool
	// RejectionCode is the kind of refusal staff picked;
	// RejectionReason the message they wrote to the owner.
	RejectionCode RejectionCode
	// Checks are the review points the staff member ticked before
	// deciding (see Catalog.FarmReviewChecks); kept as a record of the review.
	Checks          []string
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
	// RejectReverification is not a refusal of what was sent: staff ask
	// an approved farm's owner to prove their identity again.
	RejectReverification RejectionCode = "reverification"
)

// ProofPart is one piece of what an owner sends for review.
type ProofPart string

const (
	ProofIDDocument ProofPart = "id_document"
	ProofPhotos     ProofPart = "photos"
	ProofDetails    ProofPart = "details"
)

// ProofParts lists every part, in the order owners see them.
var ProofParts = []ProofPart{ProofIDDocument, ProofPhotos, ProofDetails}

// ToRedo is what a refusal asks the owner to send again. A refusal for
// another reason leaves that to the staff message: nothing is assumed
// to be fine.
func (c RejectionCode) ToRedo() []ProofPart {
	switch c {
	case RejectIDUnreadable, RejectNameMismatch, RejectReverification:
		return []ProofPart{ProofIDDocument}
	case RejectPhotosNotRecent:
		return []ProofPart{ProofPhotos}
	case RejectFarmNotFound:
		return []ProofPart{ProofDetails}
	default:
		return ProofParts
	}
}

// Accepted is the rest: what the owner need not send again.
func (c RejectionCode) Accepted() []ProofPart {
	redo := c.ToRedo()
	var kept []ProofPart
	for _, part := range ProofParts {
		found := false
		for _, r := range redo {
			found = found || r == part
		}
		if !found {
			kept = append(kept, part)
		}
	}
	return kept
}

// RejectionCodes lists the refusal reasons in the order staff see them.
var RejectionCodes = []RejectionCode{RejectIDUnreadable, RejectNameMismatch, RejectPhotosNotRecent, RejectFarmNotFound, RejectOther}

// IsValid reports whether c is a known refusal reason.
func (c RejectionCode) IsValid() bool {
	if c == RejectReverification {
		return true
	}
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
	// DefaultMaxFarmContacts applies when MAX_FARM_CONTACTS isn't set.
	DefaultMaxFarmContacts = 3
)

// FarmProofPrefix is the storage prefix a farm's proof files live
// under; keys outside it were not issued for this farm.
func FarmProofPrefix(farmID bson.ObjectID) string {
	return "farm-verifications/" + farmID.Hex() + "/"
}

// FarmBrandingPrefix is the storage prefix a farm's logo and banner
// live under; keys outside it were not issued for this farm.
func FarmBrandingPrefix(farmID bson.ObjectID) string {
	return "farms/" + farmID.Hex() + "/branding/"
}

// MaxFarmDescriptionLen bounds Farm.Description.
const MaxFarmDescriptionLen = 1000

// ValidFarmReviewChecks keeps the checks of in that are in known (the
// Catalog's review checklist), in its order.
func ValidFarmReviewChecks(in, known []string) []string {
	var out []string
	for _, known := range known {
		for _, c := range in {
			if c == known {
				out = append(out, known)
				break
			}
		}
	}
	return out
}

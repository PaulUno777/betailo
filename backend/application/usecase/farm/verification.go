package farm

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/notify"
	"github.com/pauluno777/betailo/backend/domain"
)

// ownedFarm loads a farm and confirms the caller owns it.
func ownedFarm(ctx context.Context, farms ports.FarmRepository, farmID, ownerID bson.ObjectID) (*domain.Farm, error) {
	f, err := farms.FindByID(ctx, farmID)
	if err != nil {
		return nil, err
	}
	if f.OwnerID != ownerID {
		return nil, fmt.Errorf("farm: not the owner: %w", domain.ErrForbidden)
	}
	return f, nil
}

// PresignFarmProof issues a direct-upload URL for one proof file (the
// identity document or a photo of the farm). The file never transits
// the API, and nothing but this farm's own review ever reads it.
type PresignFarmProof struct {
	Farms   ports.FarmRepository
	Storage ports.ObjectStorage
}

type PresignFarmProofOutput struct {
	Key       string
	UploadURL string
}

func (uc *PresignFarmProof) Execute(ctx context.Context, farmID, ownerID bson.ObjectID) (*PresignFarmProofOutput, error) {
	f, err := ownedFarm(ctx, uc.Farms, farmID, ownerID)
	if err != nil {
		return nil, err
	}
	key := domain.FarmProofPrefix(f.ID) + bson.NewObjectID().Hex() + ".jpg"
	uploadURL, err := uc.Storage.PresignUpload(ctx, key)
	if err != nil {
		return nil, err
	}
	return &PresignFarmProofOutput{Key: key, UploadURL: uploadURL}, nil
}

// SubmitFarmForReview hands a draft or rejected farm to staff, with the
// owner's identity document and photos of the farm.
type SubmitFarmForReview struct {
	Farms         ports.FarmRepository
	Trust         ports.TrustRecomputer
	Notifications ports.NotificationRepository
	Clock         ports.Clock
	// AutoApprove is the FARM_AUTO_APPROVE test switch: the farm is
	// approved on submission, with or without proof, and marked
	// AutoApproved so staff can still review it. Never on in a
	// marketplace with real sellers.
	AutoApprove bool
}

type SubmitFarmForReviewInput struct {
	FarmID  bson.ObjectID
	OwnerID bson.ObjectID
	// IDDocumentType defaults to the national identity card.
	IDDocumentType domain.IDDocumentType
	IDDocumentKey  string
	// IDDocumentBackKey is required for a document with two sides.
	IDDocumentBackKey string
	PhotoKeys         []string
	// Certified is the owner's declaration that they run this farm and
	// the photos are recent; nothing is submitted without it.
	Certified bool
}

// Execute returns domain.ErrForbidden for someone else's farm,
// domain.ErrConflict when the farm isn't waiting on its owner (already
// pending, approved or suspended), and domain.ErrInvalid for missing
// proof or keys this farm's upload endpoint didn't issue.
func (uc *SubmitFarmForReview) Execute(ctx context.Context, in SubmitFarmForReviewInput) (*domain.Farm, error) {
	f, err := ownedFarm(ctx, uc.Farms, in.FarmID, in.OwnerID)
	if err != nil {
		return nil, err
	}
	if !f.Status.CanSubmit() {
		return nil, domain.Coded("farm.not_submittable", fmt.Errorf("farm: a %s farm can't be submitted for review: %w", f.Status, domain.ErrConflict))
	}

	docType := in.IDDocumentType
	if docType == "" {
		docType = domain.IDNationalCard
	}
	if !docType.IsValid() {
		return nil, domain.Coded("farm.id_document_type", fmt.Errorf("farm: unknown identity document %q: %w", docType, domain.ErrInvalid))
	}

	// After a refusal the owner sends again only what was refused: the
	// parts staff accepted are kept from the last submission.
	if f.Status == domain.FarmRejected {
		prev := f.Verification
		for _, part := range prev.RejectionCode.Accepted() {
			switch part {
			case domain.ProofIDDocument:
				if in.IDDocumentKey == "" && in.IDDocumentBackKey == "" && prev.IDDocumentKey != "" {
					docType, in.IDDocumentKey, in.IDDocumentBackKey = prev.IDDocumentType, prev.IDDocumentKey, prev.IDDocumentBackKey
				}
			case domain.ProofPhotos:
				if len(in.PhotoKeys) == 0 {
					in.PhotoKeys = prev.PhotoKeys
				}
			}
		}
	}

	hasProof := in.IDDocumentKey != "" || in.IDDocumentBackKey != "" || len(in.PhotoKeys) > 0
	if hasProof || !uc.AutoApprove {
		if in.IDDocumentKey == "" {
			return nil, domain.Coded("farm.id_document_required", fmt.Errorf("farm: an identity document is required: %w", domain.ErrInvalid))
		}
		if docType.HasBack() && in.IDDocumentBackKey == "" {
			return nil, domain.Coded("farm.id_document_back_required", fmt.Errorf("farm: the back of the identity document is required: %w", domain.ErrInvalid))
		}
		if !in.Certified {
			return nil, domain.Coded("farm.certification_required", fmt.Errorf("farm: the owner must certify the submission: %w", domain.ErrInvalid))
		}
		if len(in.PhotoKeys) < domain.MinFarmProofPhotos || len(in.PhotoKeys) > domain.MaxFarmProofPhotos {
			return nil, domain.CodedWith("farm.proof_photo_count", map[string]any{"min": domain.MinFarmProofPhotos, "max": domain.MaxFarmProofPhotos},
				fmt.Errorf("farm: proof photos must number between %d and %d: %w", domain.MinFarmProofPhotos, domain.MaxFarmProofPhotos, domain.ErrInvalid))
		}
		keys := append([]string{in.IDDocumentKey}, in.PhotoKeys...)
		if in.IDDocumentBackKey != "" {
			keys = append(keys, in.IDDocumentBackKey)
		}
		if err := domain.CheckOwnedKeys(domain.FarmProofPrefix(f.ID), keys); err != nil {
			return nil, err
		}
	}
	backKey := in.IDDocumentBackKey
	if !docType.HasBack() {
		backKey = ""
	}

	now := uc.Clock.Now()
	f.Verification = domain.FarmVerification{
		IDDocumentType: docType, IDDocumentKey: in.IDDocumentKey, IDDocumentBackKey: backKey, PhotoKeys: in.PhotoKeys,
		SubmittedAt: &now, Submissions: f.Verification.Submissions + 1,
	}
	f.Status = domain.FarmPending
	if uc.AutoApprove {
		f.Status = domain.FarmApproved
		f.Verification.ReviewedAt = &now
		f.Verification.AutoApproved = true
	}
	if err := uc.Farms.Update(ctx, f); err != nil {
		return nil, err
	}
	if uc.AutoApprove {
		_ = uc.Trust.Execute(ctx, f.OwnerID)
		notifyFarmDecision(ctx, uc.Notifications, now, f)
	}
	return f, nil
}

// WithdrawFarmSubmission takes a farm back out of the review queue: its
// owner changed their mind, or wants to send better proof. The farm is
// a draft again and keeps what was sent.
type WithdrawFarmSubmission struct {
	Farms ports.FarmRepository
}

// Execute returns domain.ErrForbidden for someone else's farm and
// domain.ErrConflict when the farm isn't waiting for a decision.
func (uc *WithdrawFarmSubmission) Execute(ctx context.Context, farmID, ownerID bson.ObjectID) (*domain.Farm, error) {
	f, err := ownedFarm(ctx, uc.Farms, farmID, ownerID)
	if err != nil {
		return nil, err
	}
	if f.Status != domain.FarmPending {
		return nil, domain.Coded("farm.not_pending", fmt.Errorf("farm: a %s farm has no request to withdraw: %w", f.Status, domain.ErrConflict))
	}
	f.Status = domain.FarmDraft
	f.Verification.SubmittedAt = nil
	if err := uc.Farms.Update(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

// FarmDecision is what staff decide about a farm.
type FarmDecision string

const (
	FarmDecisionApprove FarmDecision = "approve"
	FarmDecisionReject  FarmDecision = "reject"
	// FarmDecisionSuspend takes an approved farm off the marketplace.
	FarmDecisionSuspend FarmDecision = "suspend"
)

// ReviewFarm is staff's decision on a farm: approve what was submitted,
// reject it with a reason the owner reads, or suspend a farm that was
// approved.
type ReviewFarm struct {
	Farms         ports.FarmRepository
	Trust         ports.TrustRecomputer
	Notifications ports.NotificationRepository
	Clock         ports.Clock
}

type ReviewFarmInput struct {
	FarmID     bson.ObjectID
	ReviewerID bson.ObjectID
	// ReviewerRole is recorded with the decision.
	ReviewerRole domain.StaffRole
	Decision     FarmDecision
	// ReasonCode is the kind of refusal, for a rejection (RejectOther
	// when not given); Reason the message the owner reads.
	ReasonCode domain.RejectionCode
	Reason     string
	// Checks are the review points the staff member ticked.
	Checks []string
}

// Execute returns domain.ErrInvalid for an unknown decision or a
// rejection/suspension without a reason, and domain.ErrConflict when
// the decision doesn't apply to the farm's current status.
func (uc *ReviewFarm) Execute(ctx context.Context, in ReviewFarmInput) (*domain.Farm, error) {
	f, err := uc.Farms.FindByID(ctx, in.FarmID)
	if err != nil {
		return nil, err
	}
	reason := strings.TrimSpace(in.Reason)
	if utf8.RuneCountInString(reason) > domain.MaxRejectionReasonLen {
		return nil, fmt.Errorf("farm: reason must be at most %d characters: %w", domain.MaxRejectionReasonLen, domain.ErrInvalid)
	}

	var allowedFrom []domain.FarmStatus
	var next domain.FarmStatus
	switch in.Decision {
	case FarmDecisionApprove:
		// Approving again confirms a farm the test switch let through,
		// and lifts a suspension.
		allowedFrom, next = []domain.FarmStatus{domain.FarmPending, domain.FarmApproved, domain.FarmSuspended}, domain.FarmApproved
		reason = ""
	case FarmDecisionReject:
		allowedFrom, next = []domain.FarmStatus{domain.FarmPending}, domain.FarmRejected
	case FarmDecisionSuspend:
		allowedFrom, next = []domain.FarmStatus{domain.FarmApproved}, domain.FarmSuspended
	default:
		return nil, fmt.Errorf("farm: unknown decision %q: %w", in.Decision, domain.ErrInvalid)
	}
	code := domain.RejectionCode("")
	if in.Decision == FarmDecisionReject {
		code = in.ReasonCode
		if code == "" {
			code = domain.RejectOther
		}
		if !code.IsValid() {
			return nil, domain.Coded("farm.rejection_code", fmt.Errorf("farm: unknown refusal reason %q: %w", code, domain.ErrInvalid))
		}
	}
	if next != domain.FarmApproved && reason == "" {
		return nil, domain.Coded("farm.reason_required", fmt.Errorf("farm: a reason is required to %s a farm: %w", in.Decision, domain.ErrInvalid))
	}
	allowed := false
	for _, s := range allowedFrom {
		allowed = allowed || f.Status == s
	}
	if !allowed {
		return nil, domain.Coded("farm.decision_not_applicable", fmt.Errorf("farm: can't %s a %s farm: %w", in.Decision, f.Status, domain.ErrConflict))
	}

	now := uc.Clock.Now()
	f.Status = next
	f.Verification.ReviewedAt = &now
	f.Verification.ReviewedBy = in.ReviewerID
	f.Verification.ReviewedByRole = in.ReviewerRole
	f.Verification.AutoApproved = false
	f.Verification.RejectionCode = code
	f.Verification.RejectionReason = reason
	f.Verification.Checks = domain.ValidFarmReviewChecks(in.Checks)
	if err := uc.Farms.Update(ctx, f); err != nil {
		return nil, err
	}
	_ = uc.Trust.Execute(ctx, f.OwnerID)
	notifyFarmDecision(ctx, uc.Notifications, now, f)
	return f, nil
}

func notifyFarmDecision(ctx context.Context, outbox ports.NotificationRepository, now time.Time, f *domain.Farm) {
	n := notify.Notice{UserID: f.OwnerID, Params: map[string]string{"farm": f.Name}, Link: domain.LinkSellerFarms}
	switch f.Status {
	case domain.FarmApproved:
		n.Type, n.Fallback = domain.NotifFarmApproved, fmt.Sprintf("Votre ferme %s est validée : vous pouvez publier vos annonces.", f.Name)
	case domain.FarmRejected:
		n.Type, n.Fallback = domain.NotifFarmRejected, fmt.Sprintf("La validation de votre ferme %s a été refusée. Consultez le motif et soumettez-la de nouveau.", f.Name)
	case domain.FarmSuspended:
		n.Type, n.Fallback = domain.NotifFarmSuspended, fmt.Sprintf("Votre ferme %s est suspendue : ses annonces ne sont plus visibles.", f.Name)
	default:
		return
	}
	notify.Send(ctx, outbox, now, n)
}

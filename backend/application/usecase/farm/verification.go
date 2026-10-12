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
	FarmID        bson.ObjectID
	OwnerID       bson.ObjectID
	IDDocumentKey string
	PhotoKeys     []string
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

	hasProof := in.IDDocumentKey != "" || len(in.PhotoKeys) > 0
	if hasProof || !uc.AutoApprove {
		if in.IDDocumentKey == "" {
			return nil, domain.Coded("farm.id_document_required", fmt.Errorf("farm: an identity document is required: %w", domain.ErrInvalid))
		}
		if len(in.PhotoKeys) < domain.MinFarmProofPhotos || len(in.PhotoKeys) > domain.MaxFarmProofPhotos {
			return nil, domain.CodedWith("farm.proof_photo_count", map[string]any{"min": domain.MinFarmProofPhotos, "max": domain.MaxFarmProofPhotos},
				fmt.Errorf("farm: proof photos must number between %d and %d: %w", domain.MinFarmProofPhotos, domain.MaxFarmProofPhotos, domain.ErrInvalid))
		}
		if err := domain.CheckOwnedKeys(domain.FarmProofPrefix(f.ID), append([]string{in.IDDocumentKey}, in.PhotoKeys...)); err != nil {
			return nil, err
		}
	}

	now := uc.Clock.Now()
	f.Verification = domain.FarmVerification{IDDocumentKey: in.IDDocumentKey, PhotoKeys: in.PhotoKeys, SubmittedAt: &now}
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
	Decision   FarmDecision
	Reason     string
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
	f.Verification.AutoApproved = false
	f.Verification.RejectionReason = reason
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

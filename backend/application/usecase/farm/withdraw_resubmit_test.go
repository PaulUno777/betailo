package farm

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

func pendingFarm(farms *fakeFarms, owner bson.ObjectID, status domain.FarmStatus) *domain.Farm {
	f := &domain.Farm{ID: bson.NewObjectID(), OwnerID: owner, Name: "Élevage de Bafang", Status: status}
	key := func() string { return domain.FarmProofPrefix(f.ID) + bson.NewObjectID().Hex() + ".jpg" }
	at := time.Date(2026, 10, 10, 16, 42, 0, 0, time.UTC)
	f.Verification = domain.FarmVerification{
		IDDocumentType: domain.IDNationalCard, IDDocumentKey: key(), IDDocumentBackKey: key(),
		PhotoKeys: []string{key(), key()}, SubmittedAt: &at, Submissions: 1,
	}
	farms.byID[f.ID] = f
	return f
}

func TestWithdrawFarmSubmission(t *testing.T) {
	farms, owner := newFakeFarms(), bson.NewObjectID()
	uc := &WithdrawFarmSubmission{Farms: farms}

	f := pendingFarm(farms, owner, domain.FarmPending)
	got, err := uc.Execute(context.Background(), f.ID, owner)
	if err != nil {
		t.Fatalf("withdraw: %v", err)
	}
	if got.Status != domain.FarmDraft || got.Verification.SubmittedAt != nil || len(got.Verification.PhotoKeys) != 2 {
		t.Errorf("withdrawn farm = %s, submittedAt %v, photos %d; want a draft that keeps its proof", got.Status, got.Verification.SubmittedAt, len(got.Verification.PhotoKeys))
	}

	if _, err := uc.Execute(context.Background(), f.ID, owner); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("withdrawing a draft = %v, want a conflict", err)
	}
	other := pendingFarm(farms, owner, domain.FarmPending)
	if _, err := uc.Execute(context.Background(), other.ID, bson.NewObjectID()); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("withdrawing someone else's farm = %v, want forbidden", err)
	}
}

func TestRejectionCode_ToRedoAndAccepted(t *testing.T) {
	cases := map[domain.RejectionCode][2]int{
		domain.RejectIDUnreadable:    {1, 2},
		domain.RejectNameMismatch:    {1, 2},
		domain.RejectPhotosNotRecent: {1, 2},
		domain.RejectFarmNotFound:    {1, 2},
		domain.RejectOther:           {3, 0},
	}
	for code, want := range cases {
		if redo, kept := code.ToRedo(), code.Accepted(); len(redo) != want[0] || len(kept) != want[1] {
			t.Errorf("%s: to redo %v, accepted %v", code, redo, kept)
		}
	}
	if redo := domain.RejectIDUnreadable.ToRedo(); redo[0] != domain.ProofIDDocument {
		t.Errorf("an unreadable document asks for %v", redo)
	}
}

// After a refusal for an unreadable document, the owner sends only a new
// document: the photos staff accepted are kept.
func TestSubmitFarmForReview_ResendsOnlyWhatWasRefused(t *testing.T) {
	farms, owner := newFakeFarms(), bson.NewObjectID()
	uc := &SubmitFarmForReview{Farms: farms, Trust: &fakeTrust{}, Clock: fakeClock{now: time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)}}

	f := pendingFarm(farms, owner, domain.FarmRejected)
	f.Verification.RejectionCode = domain.RejectIDUnreadable
	front, back := domain.FarmProofPrefix(f.ID)+bson.NewObjectID().Hex()+".jpg", domain.FarmProofPrefix(f.ID)+bson.NewObjectID().Hex()+".jpg"

	got, err := uc.Execute(context.Background(), SubmitFarmForReviewInput{
		FarmID: f.ID, OwnerID: owner, IDDocumentType: domain.IDNationalCard,
		IDDocumentKey: front, IDDocumentBackKey: back, Certified: true,
	})
	if err != nil {
		t.Fatalf("resubmit with a new document only: %v", err)
	}
	v := got.Verification
	if got.Status != domain.FarmPending || v.IDDocumentKey != front || len(v.PhotoKeys) != 2 || v.Submissions != 2 {
		t.Errorf("resubmitted = %s, document %s, photos %d, submissions %d", got.Status, v.IDDocumentKey, len(v.PhotoKeys), v.Submissions)
	}

	// Refused for its photos: sending no photos is not enough.
	g := pendingFarm(farms, owner, domain.FarmRejected)
	g.Verification.RejectionCode = domain.RejectPhotosNotRecent
	if _, err := uc.Execute(context.Background(), SubmitFarmForReviewInput{FarmID: g.ID, OwnerID: owner, Certified: true}); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("resubmitting refused photos without new ones = %v, want invalid", err)
	}
}

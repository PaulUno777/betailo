// Package review implements the one rule the architecture document
// insists on: a review can only ever follow a real, closed reservation
// — never freely posted. See domain.Review's doc comment.
package review

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/notify"
	"github.com/pauluno777/betailo/backend/domain"
)

// CreateReview is the usecase: verify the caller is the reservation's
// buyer and its escrow has released, resolve the farm behind it (for
// the denormalized FarmID — see domain.Review), insert, then trigger
// the farm owner's trust-score recomputation.
type CreateReview struct {
	Reservations ports.ReservationRepository
	Listings     ports.ListingStore
	Cycles       ports.CycleStore
	Farms        ports.FarmRepository
	Reviews      ports.ReviewRepository
	Trust        ports.TrustRecomputer
	Clock        ports.Clock
	// Notifications tells the farmer about the new review. Optional.
	Notifications ports.NotificationRepository
}

type CreateReviewInput struct {
	ReservationID bson.ObjectID
	AuthorID      bson.ObjectID
	Rating        int
	Comment       string
	Criteria      domain.ReviewCriteria
	Tags          []string
	// PhotoKeys come from PresignReviewPhoto for this reservation.
	PhotoKeys []string
}

// Execute returns domain.ErrForbidden if the caller isn't the
// reservation's buyer, domain.ErrConflict if the escrow hasn't released
// yet (or a review already exists for it — the reviews collection's
// unique index on reservationId is the actual enforcement), and
// domain.ErrInvalid for a rating outside 1-5.
func (uc *CreateReview) Execute(ctx context.Context, in CreateReviewInput) (*domain.Review, error) {
	if in.Rating < 1 || in.Rating > 5 {
		return nil, domain.Coded("review.rating_range", fmt.Errorf("review: rating must be between 1 and 5: %w", domain.ErrInvalid))
	}
	if err := in.Criteria.Validate(); err != nil {
		return nil, err
	}
	tags, err := domain.NormalizeReviewTags(in.Tags)
	if err != nil {
		return nil, err
	}
	comment, err := domain.ValidateReviewText(in.Comment, domain.MaxReviewCommentLen)
	if err != nil {
		return nil, err
	}
	if len(in.PhotoKeys) > domain.MaxReviewPhotos {
		return nil, fmt.Errorf("review: at most %d photos: %w", domain.MaxReviewPhotos, domain.ErrInvalid)
	}
	if err := domain.CheckOwnedKeys(PhotoPrefix(in.ReservationID), in.PhotoKeys); err != nil {
		return nil, err
	}

	res, err := uc.Reservations.FindByID(ctx, in.ReservationID)
	if err != nil {
		return nil, err
	}
	if res.BuyerID != in.AuthorID {
		return nil, fmt.Errorf("review: not the reservation's buyer: %w", domain.ErrForbidden)
	}
	if res.Payment == nil || res.Payment.Status != domain.PaymentReleased {
		return nil, domain.Coded("review.not_released", fmt.Errorf("review: only reviewable once escrow has released: %w", domain.ErrConflict))
	}
	if done, err := uc.Reviews.ExistsForReservation(ctx, res.ID); err != nil {
		return nil, err
	} else if done {
		return nil, domain.Coded("review.already_reviewed", fmt.Errorf("review: this reservation is already reviewed: %w", domain.ErrConflict))
	}

	l, err := uc.Listings.FindByID(ctx, res.ListingID)
	if err != nil {
		return nil, err
	}
	if l.CycleID == nil {
		return nil, fmt.Errorf("review: listing has no cycle: %w", domain.ErrInvalid)
	}
	c, err := uc.Cycles.FindByID(ctx, *l.CycleID)
	if err != nil {
		return nil, err
	}

	rv := &domain.Review{
		ID:            bson.NewObjectID(),
		ReservationID: in.ReservationID,
		AuthorID:      in.AuthorID,
		FarmID:        c.FarmID,
		Rating:        in.Rating,
		Comment:       comment,
		Criteria:      in.Criteria,
		Tags:          tags,
		Photos:        in.PhotoKeys,
		CreatedAt:     uc.now(),
	}
	if err := uc.Reviews.Insert(ctx, rv); err != nil {
		return nil, err
	}

	if f, err := uc.Farms.FindByID(ctx, c.FarmID); err == nil {
		_ = uc.Trust.Execute(ctx, f.OwnerID)
		if uc.Clock != nil {
			notify.Send(ctx, uc.Notifications, uc.Clock.Now(), notify.Notice{
				UserID: f.OwnerID, Type: domain.NotifReviewReceived,
				Params: map[string]string{"rating": strconv.Itoa(in.Rating), "ref": res.Reference},
				Link:   "/portal", Fallback: fmt.Sprintf("Nouvel avis %d/5 sur %s.", in.Rating, res.Reference),
			})
		}
	}

	return rv, nil
}

func (uc *CreateReview) now() time.Time {
	if uc.Clock == nil {
		return time.Now()
	}
	return uc.Clock.Now()
}

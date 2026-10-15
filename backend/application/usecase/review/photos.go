package review

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// PhotoPrefix is where a reservation's review photos live in storage.
func PhotoPrefix(reservationID bson.ObjectID) string {
	return "reviews/" + reservationID.Hex() + "/"
}

// PresignReviewPhoto hands the buyer an upload URL for one review photo,
// under the same conditions as reviewing: their reservation, escrow
// released.
type PresignReviewPhoto struct {
	Reservations ports.ReservationRepository
	Storage      ports.ObjectStorage
}

type PresignReviewPhotoOutput struct {
	Key            string
	UploadURL      string
	ThumbUploadURL string
}

// Execute locks the photo's upload URL to size bytes and its small
// copy's to thumbSize, each under its cap.
func (uc *PresignReviewPhoto) Execute(ctx context.Context, reservationID, buyerID bson.ObjectID, size, thumbSize int64) (*PresignReviewPhotoOutput, error) {
	if err := domain.CheckPhotoSize(size, domain.MaxPhotoBytes); err != nil {
		return nil, err
	}
	if err := domain.CheckPhotoSize(thumbSize, domain.MaxThumbBytes); err != nil {
		return nil, err
	}
	res, err := uc.Reservations.FindByID(ctx, reservationID)
	if err != nil {
		return nil, err
	}
	if res.BuyerID != buyerID {
		return nil, fmt.Errorf("review: not the reservation's buyer: %w", domain.ErrForbidden)
	}
	if res.Payment == nil || res.Payment.Status != domain.PaymentReleased {
		return nil, domain.Coded("review.not_released", fmt.Errorf("review: only reviewable once escrow has released: %w", domain.ErrConflict))
	}
	key := PhotoPrefix(reservationID) + bson.NewObjectID().Hex() + ".jpg"
	url, err := uc.Storage.PresignUpload(ctx, key, size)
	if err != nil {
		return nil, err
	}
	thumbURL, err := uc.Storage.PresignUpload(ctx, domain.ThumbKey(key), thumbSize)
	if err != nil {
		return nil, err
	}
	return &PresignReviewPhotoOutput{Key: key, UploadURL: url, ThumbUploadURL: thumbURL}, nil
}

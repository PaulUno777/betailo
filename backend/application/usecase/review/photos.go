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
	Key       string
	UploadURL string
}

func (uc *PresignReviewPhoto) Execute(ctx context.Context, reservationID, buyerID bson.ObjectID) (*PresignReviewPhotoOutput, error) {
	res, err := uc.Reservations.FindByID(ctx, reservationID)
	if err != nil {
		return nil, err
	}
	if res.BuyerID != buyerID {
		return nil, fmt.Errorf("review: not the reservation's buyer: %w", domain.ErrForbidden)
	}
	if res.Payment == nil || res.Payment.Status != domain.PaymentReleased {
		return nil, fmt.Errorf("review: only reviewable once escrow has released: %w", domain.ErrConflict)
	}
	key := PhotoPrefix(reservationID) + bson.NewObjectID().Hex() + ".jpg"
	url, err := uc.Storage.PresignUpload(ctx, key)
	if err != nil {
		return nil, err
	}
	return &PresignReviewPhotoOutput{Key: key, UploadURL: url}, nil
}

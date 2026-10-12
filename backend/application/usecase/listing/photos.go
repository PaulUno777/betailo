package listing

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// MinListingPhotos/MaxListingPhotos match the design's "Photos · 3 à 12"
// requirement, enforced here — server-side — rather than left to the
// publish form's own validation.
// A listing can be published with a single photo: sellers work from
// their phones, and asking for three before anything goes live made the
// sell flow stall.
const (
	MinListingPhotos = 1
	MaxListingPhotos = 12
)

// PresignListingPhoto issues a direct-upload URL for one photo — the
// file itself never transits the API server (see
// ports.ObjectStorage.PresignUpload's doc comment).
type PresignListingPhoto struct {
	Farms    ports.FarmRepository
	Cycles   ports.CycleStore
	Listings ports.ListingStore
	Storage  ports.ObjectStorage
}

type PresignListingPhotoInput struct {
	ListingID bson.ObjectID
	OwnerID   bson.ObjectID
}

type PresignListingPhotoOutput struct {
	Key       string
	UploadURL string
}

// Execute returns domain.ErrForbidden if the caller doesn't own the
// listing's cycle's farm.
func (uc *PresignListingPhoto) Execute(ctx context.Context, in PresignListingPhotoInput) (*PresignListingPhotoOutput, error) {
	l, err := uc.Listings.FindByID(ctx, in.ListingID)
	if err != nil {
		return nil, err
	}
	actualOwner, err := ownerOf(ctx, uc.Cycles, uc.Farms, l)
	if err != nil {
		return nil, err
	}
	if actualOwner != in.OwnerID {
		return nil, fmt.Errorf("listing: not the owner: %w", domain.ErrForbidden)
	}

	key := ListingPhotoPrefix(in.ListingID) + bson.NewObjectID().Hex() + ".jpg"
	uploadURL, err := uc.Storage.PresignUpload(ctx, key)
	if err != nil {
		return nil, err
	}
	return &PresignListingPhotoOutput{Key: key, UploadURL: uploadURL}, nil
}

// SetListingPhotos replaces a listing's ordered photo list — the first
// key is the cover, matching the design's publish-form preview.
// ListingPhotoPrefix is where a listing's photos live in object storage.
func ListingPhotoPrefix(listingID bson.ObjectID) string {
	return "listings/" + listingID.Hex() + "/"
}

type SetListingPhotos struct {
	Farms    ports.FarmRepository
	Cycles   ports.CycleStore
	Listings ports.ListingStore
}

type SetListingPhotosInput struct {
	ListingID bson.ObjectID
	OwnerID   bson.ObjectID
	Keys      []string
}

// Execute returns domain.ErrForbidden if the caller doesn't own the
// listing, and domain.ErrInvalid if the key count is outside
// [MinListingPhotos, MaxListingPhotos] — the design's "Contrôles avant
// publication" checklist, enforced here rather than only in the form.
func (uc *SetListingPhotos) Execute(ctx context.Context, in SetListingPhotosInput) (*domain.Listing, error) {
	if len(in.Keys) < MinListingPhotos || len(in.Keys) > MaxListingPhotos {
		return nil, fmt.Errorf("listing: photos must number between %d and %d: %w", MinListingPhotos, MaxListingPhotos, domain.ErrInvalid)
	}

	l, err := uc.Listings.FindByID(ctx, in.ListingID)
	if err != nil {
		return nil, err
	}
	actualOwner, err := ownerOf(ctx, uc.Cycles, uc.Farms, l)
	if err != nil {
		return nil, err
	}
	if actualOwner != in.OwnerID {
		return nil, fmt.Errorf("listing: not the owner: %w", domain.ErrForbidden)
	}
	if err := domain.CheckOwnedKeys(ListingPhotoPrefix(in.ListingID), in.Keys); err != nil {
		return nil, err
	}

	l.Photos = in.Keys
	if err := uc.Listings.Update(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

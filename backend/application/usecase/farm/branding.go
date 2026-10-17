package farm

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// BrandingImage is which of a farm's two public images is meant.
type BrandingImage string

const (
	// BrandingLogo is the square mark shown beside the farm's name.
	BrandingLogo BrandingImage = "logo"
	// BrandingBanner is the wide picture across the top of its card.
	BrandingBanner BrandingImage = "banner"
)

// PresignBranding hands the owner upload URLs for a new logo or banner
// and its small copy, locked to the sizes the browser declared.
type PresignBranding struct {
	Farms   ports.FarmRepository
	Storage ports.ObjectStorage
}

type PresignBrandingOutput struct {
	Key            string
	UploadURL      string
	ThumbUploadURL string
}

// Execute returns domain.ErrInvalid if either size is over its cap and
// domain.ErrForbidden if ownerID doesn't own the farm.
func (uc *PresignBranding) Execute(ctx context.Context, farmID, ownerID bson.ObjectID, size, thumbSize int64) (*PresignBrandingOutput, error) {
	if err := domain.CheckPhotoSize(size, domain.MaxPhotoBytes); err != nil {
		return nil, err
	}
	if err := domain.CheckPhotoSize(thumbSize, domain.MaxThumbBytes); err != nil {
		return nil, err
	}
	f, err := ownedFarm(ctx, uc.Farms, farmID, ownerID)
	if err != nil {
		return nil, err
	}
	key := domain.FarmBrandingPrefix(f.ID) + bson.NewObjectID().Hex() + ".jpg"
	uploadURL, err := uc.Storage.PresignUpload(ctx, key, size)
	if err != nil {
		return nil, err
	}
	thumbURL, err := uc.Storage.PresignUpload(ctx, domain.ThumbKey(key), thumbSize)
	if err != nil {
		return nil, err
	}
	return &PresignBrandingOutput{Key: key, UploadURL: uploadURL, ThumbUploadURL: thumbURL}, nil
}

// SetBranding makes an uploaded image the farm's logo or banner, or
// removes it when key is empty (the app then shows its default).
type SetBranding struct {
	Farms ports.FarmRepository
}

// Execute returns domain.ErrInvalid for an unknown image or a key that
// PresignBranding didn't mint for this farm, and domain.ErrForbidden
// if ownerID doesn't own it.
func (uc *SetBranding) Execute(ctx context.Context, farmID, ownerID bson.ObjectID, image BrandingImage, key string) (*domain.Farm, error) {
	if image != BrandingLogo && image != BrandingBanner {
		return nil, fmt.Errorf("farm: unknown branding image %q: %w", image, domain.ErrInvalid)
	}
	f, err := ownedFarm(ctx, uc.Farms, farmID, ownerID)
	if err != nil {
		return nil, err
	}
	if key != "" {
		if err := domain.CheckOwnedKeys(domain.FarmBrandingPrefix(f.ID), []string{key}); err != nil {
			return nil, err
		}
	}
	if image == BrandingLogo {
		f.LogoKey = key
	} else {
		f.BannerKey = key
	}
	if err := uc.Farms.Update(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

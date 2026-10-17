package farm

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

type fakeStorage struct {
	ports.ObjectStorage
	signed map[string]int64
}

func (f *fakeStorage) PresignUpload(_ context.Context, key string, size int64) (string, error) {
	f.signed[key] = size
	return "https://storage.test/" + key, nil
}

func seedFarm(t *testing.T, farms *fakeFarms, owner bson.ObjectID) *domain.Farm {
	t.Helper()
	f := &domain.Farm{OwnerID: owner, Name: "Ferme", Location: "Douala"}
	if err := farms.Create(context.Background(), f); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	return f
}

func TestPresignBranding_LocksBothUploadsUnderTheFarm(t *testing.T) {
	farms := newFakeFarms()
	owner := bson.NewObjectID()
	f := seedFarm(t, farms, owner)
	storage := &fakeStorage{signed: map[string]int64{}}
	uc := &PresignBranding{Farms: farms, Storage: storage}

	out, err := uc.Execute(context.Background(), f.ID, owner, 300<<10, 40<<10)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.HasPrefix(out.Key, domain.FarmBrandingPrefix(f.ID)) {
		t.Errorf("key %q is outside the farm's branding prefix", out.Key)
	}
	if storage.signed[out.Key] != 300<<10 || storage.signed[domain.ThumbKey(out.Key)] != 40<<10 {
		t.Errorf("signed sizes = %v", storage.signed)
	}

	if _, err := uc.Execute(context.Background(), f.ID, owner, domain.MaxPhotoBytes+1, 40<<10); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("oversized image: error = %v, want ErrInvalid", err)
	}
	if _, err := uc.Execute(context.Background(), f.ID, bson.NewObjectID(), 300<<10, 40<<10); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("not the owner: error = %v, want ErrForbidden", err)
	}
}

func TestSetBranding_SetsAndClearsEachImage(t *testing.T) {
	farms := newFakeFarms()
	owner := bson.NewObjectID()
	f := seedFarm(t, farms, owner)
	uc := &SetBranding{Farms: farms}
	logo := domain.FarmBrandingPrefix(f.ID) + bson.NewObjectID().Hex() + ".jpg"
	banner := domain.FarmBrandingPrefix(f.ID) + bson.NewObjectID().Hex() + ".jpg"

	if _, err := uc.Execute(context.Background(), f.ID, owner, BrandingLogo, logo); err != nil {
		t.Fatalf("set logo: %v", err)
	}
	got, err := uc.Execute(context.Background(), f.ID, owner, BrandingBanner, banner)
	if err != nil {
		t.Fatalf("set banner: %v", err)
	}
	if got.LogoKey != logo || got.BannerKey != banner {
		t.Errorf("keys = %q, %q; want %q, %q", got.LogoKey, got.BannerKey, logo, banner)
	}

	got, err = uc.Execute(context.Background(), f.ID, owner, BrandingLogo, "")
	if err != nil || got.LogoKey != "" || got.BannerKey != banner {
		t.Errorf("clearing the logo = %+v, %v; want only the logo removed", got, err)
	}
}

func TestSetBranding_Refuses(t *testing.T) {
	farms := newFakeFarms()
	owner := bson.NewObjectID()
	f := seedFarm(t, farms, owner)
	other := seedFarm(t, farms, owner)
	uc := &SetBranding{Farms: farms}
	own := domain.FarmBrandingPrefix(f.ID) + bson.NewObjectID().Hex() + ".jpg"

	tests := []struct {
		name  string
		owner bson.ObjectID
		image BrandingImage
		key   string
		want  error
	}{
		{"another farm's key", owner, BrandingLogo, domain.FarmBrandingPrefix(other.ID) + bson.NewObjectID().Hex() + ".jpg", domain.ErrInvalid},
		{"a proof key", owner, BrandingBanner, domain.FarmProofPrefix(f.ID) + bson.NewObjectID().Hex() + ".jpg", domain.ErrInvalid},
		{"unknown image", owner, "cover", own, domain.ErrInvalid},
		{"not the owner", bson.NewObjectID(), BrandingLogo, own, domain.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := uc.Execute(context.Background(), f.ID, tt.owner, tt.image, tt.key); !errors.Is(err, tt.want) {
				t.Errorf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

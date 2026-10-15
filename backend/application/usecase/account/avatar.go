package account

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// AvatarPrefix is where an account's profile photos live. A key outside
// it can't be set as that account's photo.
func AvatarPrefix(userID bson.ObjectID) string {
	return "avatars/" + userID.Hex() + "/"
}

// PresignAvatar hands out upload URLs for a new profile photo and its
// small copy, locked to the sizes the browser declared.
type PresignAvatar struct {
	Storage ports.ObjectStorage
}

type PresignAvatarOutput struct {
	Key            string
	UploadURL      string
	ThumbUploadURL string
}

// Execute returns domain.ErrInvalid if either size is over its cap.
func (uc *PresignAvatar) Execute(ctx context.Context, userID bson.ObjectID, size, thumbSize int64) (*PresignAvatarOutput, error) {
	if err := domain.CheckPhotoSize(size, domain.MaxPhotoBytes); err != nil {
		return nil, err
	}
	if err := domain.CheckPhotoSize(thumbSize, domain.MaxThumbBytes); err != nil {
		return nil, err
	}
	key := AvatarPrefix(userID) + bson.NewObjectID().Hex() + ".jpg"
	uploadURL, err := uc.Storage.PresignUpload(ctx, key, size)
	if err != nil {
		return nil, err
	}
	thumbURL, err := uc.Storage.PresignUpload(ctx, domain.ThumbKey(key), thumbSize)
	if err != nil {
		return nil, err
	}
	return &PresignAvatarOutput{Key: key, UploadURL: uploadURL, ThumbUploadURL: thumbURL}, nil
}

// SetAvatar makes an uploaded photo the account's profile photo, or
// removes it when key is empty.
type SetAvatar struct {
	Users ports.UserRepository
}

// Execute returns domain.ErrInvalid for a key that wasn't minted for
// this account by PresignAvatar.
func (uc *SetAvatar) Execute(ctx context.Context, userID bson.ObjectID, key string) (*domain.User, error) {
	if key != "" {
		if err := domain.CheckOwnedKeys(AvatarPrefix(userID), []string{key}); err != nil {
			return nil, err
		}
	}
	u, err := uc.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	u.AvatarKey = key
	if err := uc.Users.Update(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

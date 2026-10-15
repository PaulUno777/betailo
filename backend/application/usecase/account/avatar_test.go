package account

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

type fakeUsers struct {
	ports.UserRepository
	byID map[bson.ObjectID]*domain.User
}

func (f *fakeUsers) FindByID(_ context.Context, id bson.ObjectID) (*domain.User, error) {
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return nil, domain.ErrNotFound
}

func (f *fakeUsers) Update(_ context.Context, u *domain.User) error {
	f.byID[u.ID] = u
	return nil
}

type fakeStorage struct {
	ports.ObjectStorage
	signed map[string]int64
}

func (f *fakeStorage) PresignUpload(_ context.Context, key string, size int64) (string, error) {
	f.signed[key] = size
	return "https://storage.test/" + key, nil
}

func TestPresignAvatar_LocksBothUploadsUnderTheAccount(t *testing.T) {
	userID := bson.NewObjectID()
	storage := &fakeStorage{signed: map[string]int64{}}

	out, err := (&PresignAvatar{Storage: storage}).Execute(context.Background(), userID, 300<<10, 40<<10)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.HasPrefix(out.Key, AvatarPrefix(userID)) {
		t.Errorf("key %q is outside the account's prefix", out.Key)
	}
	if storage.signed[out.Key] != 300<<10 || storage.signed[domain.ThumbKey(out.Key)] != 40<<10 {
		t.Errorf("signed sizes = %v", storage.signed)
	}
}

func TestPresignAvatar_RefusesOversizedPhotos(t *testing.T) {
	uc := &PresignAvatar{Storage: &fakeStorage{signed: map[string]int64{}}}
	for _, sizes := range [][2]int64{{domain.MaxPhotoBytes + 1, 1000}, {1000, domain.MaxThumbBytes + 1}, {0, 1000}} {
		if _, err := uc.Execute(context.Background(), bson.NewObjectID(), sizes[0], sizes[1]); !errors.Is(err, domain.ErrInvalid) {
			t.Errorf("sizes %v: err = %v, want ErrInvalid", sizes, err)
		}
	}
}

func TestSetAvatar(t *testing.T) {
	me, other := bson.NewObjectID(), bson.NewObjectID()
	mine := AvatarPrefix(me) + bson.NewObjectID().Hex() + ".jpg"
	users := &fakeUsers{byID: map[bson.ObjectID]*domain.User{me: {ID: me}}}
	uc := &SetAvatar{Users: users}

	u, err := uc.Execute(context.Background(), me, mine)
	if err != nil || u.AvatarKey != mine {
		t.Fatalf("set own photo: key %q, err %v", u.AvatarKey, err)
	}

	for _, key := range []string{
		AvatarPrefix(other) + bson.NewObjectID().Hex() + ".jpg",                          // someone else's upload
		"listings/" + bson.NewObjectID().Hex() + "/" + bson.NewObjectID().Hex() + ".jpg", // another kind of photo
		AvatarPrefix(me) + "../x.jpg",
	} {
		if _, err := uc.Execute(context.Background(), me, key); !errors.Is(err, domain.ErrInvalid) {
			t.Errorf("key %q: err = %v, want ErrInvalid", key, err)
		}
	}
	if users.byID[me].AvatarKey != mine {
		t.Errorf("a refused key replaced the photo: %q", users.byID[me].AvatarKey)
	}

	u, err = uc.Execute(context.Background(), me, "")
	if err != nil || u.AvatarKey != "" {
		t.Fatalf("remove photo: key %q, err %v", u.AvatarKey, err)
	}
}

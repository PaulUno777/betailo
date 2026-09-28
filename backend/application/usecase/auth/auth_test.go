package auth

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// fakeUsers, fakeHasher, fakeSigner and fakeClock are in-memory doubles
// for the ports Register and Login depend on — the point of defining
// those ports in application/ports is exactly this: both usecases are
// tested here without a MongoDB instance, bcrypt, or a JWT library.

type fakeUsers struct {
	byEmail map[string]*domain.User
	// createErr, if set, is returned by Create instead of inserting —
	// used to simulate the unique-index violation a real repository
	// would surface as domain.ErrEmailTaken.
	createErr error
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{byEmail: map[string]*domain.User{}}
}

func (f *fakeUsers) Create(_ context.Context, u *domain.User) error {
	if f.createErr != nil {
		return f.createErr
	}
	if _, exists := f.byEmail[u.Email]; exists {
		return domain.ErrEmailTaken
	}
	u.ID = bson.NewObjectID()
	f.byEmail[u.Email] = u
	return nil
}

func (f *fakeUsers) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := f.byEmail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (f *fakeUsers) FindByID(_ context.Context, id bson.ObjectID) (*domain.User, error) {
	for _, u := range f.byEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeUsers) Update(_ context.Context, u *domain.User) error {
	f.byEmail[u.Email] = u
	return nil
}

func (f *fakeUsers) List(_ context.Context) ([]*domain.User, error) {
	out := make([]*domain.User, 0, len(f.byEmail))
	for _, u := range f.byEmail {
		out = append(out, u)
	}
	return out, nil
}

// fakeHasher is intentionally not bcrypt — it just needs Hash/Compare
// to agree with each other, so the usecase's own logic is what's under
// test, not bcrypt's.
type fakeHasher struct{}

func (fakeHasher) Hash(password string) (string, error) {
	return "hashed:" + password, nil
}

func (fakeHasher) Compare(hash, password string) error {
	if hash != "hashed:"+password {
		return domain.ErrUnauthorized
	}
	return nil
}

type fakeSigner struct {
	signed []ports.TokenClaims // records each SignAccessToken call's claims, for assertions
}

func (f *fakeSigner) SignAccessToken(_ context.Context, claims ports.TokenClaims) (string, error) {
	f.signed = append(f.signed, claims)
	return "signed-token", nil
}

func (f *fakeSigner) ParseAccessToken(_ context.Context, token string) (ports.TokenClaims, error) {
	for _, c := range f.signed {
		if token == "signed-token" {
			return c, nil
		}
	}
	return ports.TokenClaims{}, domain.ErrUnauthorized
}

type fakeClock struct {
	now time.Time
}

func (f fakeClock) Now() time.Time { return f.now }

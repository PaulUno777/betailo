package httpserver

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"

	"github.com/pauluno777/betailo/backend/application/usecase/auth"
	"github.com/pauluno777/betailo/backend/application/usecase/cycle"
	"github.com/pauluno777/betailo/backend/application/usecase/farm"
	"github.com/pauluno777/betailo/backend/application/usecase/listing"
	"github.com/pauluno777/betailo/backend/domain"
	"github.com/pauluno777/betailo/backend/infrastructure/clock"
	"github.com/pauluno777/betailo/backend/infrastructure/jwt"
	"github.com/pauluno777/betailo/backend/infrastructure/security"
)

// These HTTP tests exercise the real request pipeline — JSON decoding,
// the real usecases, real bcrypt hashing and real JWT signing/parsing —
// with only persistence faked (an in-memory map instead of MongoDB).
// That's deliberate: it's the wiring in server.go, the middleware, and
// the handlers' status-code/JSON-shape behavior that's under test here,
// on top of infrastructure that's already covered on its own
// (infrastructure/security, infrastructure/jwt) and usecases already
// covered with fakes (application/usecase/auth) — this is the layer
// none of those touch.

type fakeUserRepo struct {
	byEmail map[string]*domain.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byEmail: map[string]*domain.User{}}
}

func (f *fakeUserRepo) Create(_ context.Context, u *domain.User) error {
	if _, exists := f.byEmail[u.Email]; exists {
		return domain.ErrEmailTaken
	}
	if u.ID.IsZero() {
		u.ID = bson.NewObjectID()
	}
	f.byEmail[u.Email] = u
	return nil
}

func (f *fakeUserRepo) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := f.byEmail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

// newTestRouter builds a full router with real bcrypt (at the fast
// MinCost, so tests stay quick), a real JWT signer, and the real system
// clock — the signer's exp claim is validated against actual wall-clock
// time by the JWT library regardless of what Clock a usecase used to
// compute it, so these HTTP tests can't use a fixed/fake clock the way
// application/usecase/auth's own tests do. Persistence is the only
// thing faked, via an in-memory user store.
func newTestRouter() http.Handler {
	users := newFakeUserRepo()
	farms := newFakeFarmRepo()
	cycles := newFakeCycleStore()
	listings := newFakeListingStore()
	hasher := security.BcryptHasher{Cost: bcrypt.MinCost}
	signer := &jwt.Signer{Secret: []byte("test-secret-do-not-use-in-production")}
	systemClock := clock.System{}

	deps := Dependencies{
		Register: &auth.Register{Users: users, Hasher: hasher, Clock: systemClock},
		Login:    &auth.Login{Users: users, Hasher: hasher, Signer: signer, Clock: systemClock, TTL: time.Hour},
		Signer:   signer,

		CreateFarm:  &farm.CreateFarm{Farms: farms, Clock: systemClock},
		GetFarm:     &farm.GetFarm{Farms: farms},
		ListMyFarms: &farm.ListMyFarms{Farms: farms},
		UpdateFarm:  &farm.UpdateFarm{Farms: farms},
		DeleteFarm:  &farm.DeleteFarm{Farms: farms, Cycles: cycles},

		CreateCycle:       &cycle.CreateCycle{Farms: farms, Cycles: cycles},
		GetCycle:          &cycle.GetCycle{Cycles: cycles},
		ListCyclesByFarm:  &cycle.ListCyclesByFarm{Cycles: cycles},
		UpdateCycleStatus: &cycle.UpdateCycleStatus{Farms: farms, Cycles: cycles},
		DeleteCycle:       &cycle.DeleteCycle{Farms: farms, Cycles: cycles, Listings: listings},

		CreateListing:         &listing.CreateListing{Farms: farms, Cycles: cycles, Listings: listings},
		GetListing:            &listing.GetListing{Listings: listings},
		ListPublishedListings: &listing.ListPublishedListings{Listings: listings},
		UpdateListing:         &listing.UpdateListing{Farms: farms, Cycles: cycles, Listings: listings},
		UnpublishListing:      &listing.UnpublishListing{Farms: farms, Cycles: cycles, Listings: listings},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewRouter(logger, deps)
}

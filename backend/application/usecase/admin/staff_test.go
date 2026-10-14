package admin

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

type fakeUsers struct {
	byID map[bson.ObjectID]*domain.User
}

func newFakeUsers(users ...*domain.User) *fakeUsers {
	f := &fakeUsers{byID: map[bson.ObjectID]*domain.User{}}
	for _, u := range users {
		if u.ID.IsZero() {
			u.ID = bson.NewObjectID()
		}
		f.byID[u.ID] = u
	}
	return f
}

func (f *fakeUsers) Create(_ context.Context, u *domain.User) error {
	u.ID = bson.NewObjectID()
	f.byID[u.ID] = u
	return nil
}

func (f *fakeUsers) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, u := range f.byID {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeUsers) FindByID(_ context.Context, id bson.ObjectID) (*domain.User, error) {
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return nil, domain.ErrNotFound
}

func (f *fakeUsers) List(context.Context) ([]*domain.User, error) {
	out := make([]*domain.User, 0, len(f.byID))
	for _, u := range f.byID {
		out = append(out, u)
	}
	return out, nil
}

func (f *fakeUsers) Update(_ context.Context, u *domain.User) error {
	f.byID[u.ID] = u
	return nil
}

type fakeFarms struct{ farms []*domain.Farm }

func (f *fakeFarms) Create(context.Context, *domain.Farm) error { return nil }
func (f *fakeFarms) FindByID(context.Context, bson.ObjectID) (*domain.Farm, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeFarms) ListByOwner(_ context.Context, ownerID bson.ObjectID) ([]*domain.Farm, error) {
	var out []*domain.Farm
	for _, farm := range f.farms {
		if farm.OwnerID == ownerID {
			out = append(out, farm)
		}
	}
	return out, nil
}
func (f *fakeFarms) ListByStatus(_ context.Context, status domain.FarmStatus) ([]*domain.Farm, error) {
	var out []*domain.Farm
	for _, farm := range f.farms {
		if status == "" || farm.Status == status {
			out = append(out, farm)
		}
	}
	return out, nil
}
func (f *fakeFarms) Update(context.Context, *domain.Farm) error  { return nil }
func (f *fakeFarms) Delete(context.Context, bson.ObjectID) error { return nil }

type fakeClock struct{ now time.Time }

func (c fakeClock) Now() time.Time { return c.now }

type forgotten struct{ ids []bson.ObjectID }

func (f *forgotten) Forget(id bson.ObjectID) { f.ids = append(f.ids, id) }

func TestSetStaffRole_TheLastSuperAdminKeepsTheRole(t *testing.T) {
	last := &domain.User{Email: "last@betailo.test", StaffRole: domain.StaffSuperAdmin}
	suspended := &domain.User{Email: "gone@betailo.test", StaffRole: domain.StaffSuperAdmin, Suspended: true}
	moderator := &domain.User{Email: "mod@betailo.test", StaffRole: domain.StaffModerator}
	users := newFakeUsers(last, suspended, moderator)
	uc := &SetStaffRole{Users: users, Farms: &fakeFarms{}}

	// Whoever asks: a suspended super admin doesn't count as another one.
	_, err := uc.Execute(context.Background(), SetStaffRoleInput{Actor: Actor{ID: suspended.ID, Role: domain.StaffSuperAdmin}, UserID: last.ID, Role: domain.StaffSupport})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("demoting the last active super admin: error = %v, want domain.ErrConflict", err)
	}
	if code, _ := domain.CodeOf(err); code != "admin.last_super_admin" {
		t.Errorf("code = %q, want admin.last_super_admin", code)
	}
	if last.StaffRole != domain.StaffSuperAdmin {
		t.Error("the role changed despite the refusal")
	}
}

func TestSetStaffRole_GrantClearsASellerLimitAndEndsCachedSessions(t *testing.T) {
	actor := &domain.User{Email: "root@betailo.test", StaffRole: domain.StaffSuperAdmin}
	member := &domain.User{Email: "rita@betailo.test", FarmLimit: 3, FarmLimitNote: &domain.FarmLimitNote{Reason: "coop"}}
	users := newFakeUsers(actor, member)
	sessions := &forgotten{}
	uc := &SetStaffRole{Users: users, Farms: &fakeFarms{}, Sessions: sessions}

	out, err := uc.Execute(context.Background(), SetStaffRoleInput{Actor: Actor{ID: actor.ID, Role: actor.StaffRole}, Email: " Rita@Betailo.test ", Role: domain.StaffFinance})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if out.Previous != domain.StaffNone || out.User.StaffRole != domain.StaffFinance {
		t.Fatalf("out = previous %q, role %q", out.Previous, out.User.StaffRole)
	}
	if member.FarmLimit != 0 || member.FarmLimitNote != nil {
		t.Error("a staff account kept a seller's farm limit")
	}
	if len(sessions.ids) != 1 || sessions.ids[0] != member.ID {
		t.Errorf("sessions forgotten = %v, want the member's", sessions.ids)
	}
}

func TestSetFarmLimit_RefusesStaffAndUnknownAccounts(t *testing.T) {
	staff := &domain.User{Email: "mod@betailo.test", StaffRole: domain.StaffModerator}
	users := newFakeUsers(staff)
	uc := &SetFarmLimit{Users: users, Farms: &fakeFarms{}, Clock: fakeClock{now: time.Now()}}

	_, err := uc.Execute(context.Background(), SetFarmLimitInput{UserID: staff.ID, Limit: 2, Reason: "x"})
	if code, _ := domain.CodeOf(err); code != "admin.farm_limit_staff" {
		t.Errorf("limit on a staff account: code = %q (%v), want admin.farm_limit_staff", code, err)
	}
	_, err = uc.Execute(context.Background(), SetFarmLimitInput{Email: "nobody@betailo.test", Limit: 2, Reason: "x"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("limit on an unknown account: error = %v, want domain.ErrNotFound", err)
	}
	_, err = uc.Execute(context.Background(), SetFarmLimitInput{Limit: 2, Reason: "x"})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("limit with no account: error = %v, want domain.ErrInvalid", err)
	}
}

func TestPeriodBuckets(t *testing.T) {
	now := time.Date(2026, 10, 12, 15, 30, 0, 0, time.UTC)
	tests := []struct {
		period        Period
		wantLen       int
		wantFirst     time.Time
		wantLast      time.Time
		wantPrevStart time.Time
	}{
		{Period7Days, 7, time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC), time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)},
		{Period30Days, 30, time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC), time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)},
		{Period12Month, 12, time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC)},
	}
	for _, tt := range tests {
		buckets, prev, err := periodBuckets(tt.period, now)
		if err != nil {
			t.Fatalf("%s: %v", tt.period, err)
		}
		if len(buckets) != tt.wantLen || !buckets[0].Start.Equal(tt.wantFirst) || !buckets[len(buckets)-1].Start.Equal(tt.wantLast) || !prev.Equal(tt.wantPrevStart) {
			t.Errorf("%s: %d buckets from %v to %v, previous from %v", tt.period, len(buckets), buckets[0].Start, buckets[len(buckets)-1].Start, prev)
		}
	}
	if _, _, err := periodBuckets("1y", now); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("unknown period: error = %v, want domain.ErrInvalid", err)
	}
}

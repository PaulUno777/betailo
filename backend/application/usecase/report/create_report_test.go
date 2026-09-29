package report

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/domain"
)

type fakeReports struct {
	inserted []*domain.Report
	updated  []*domain.Report
}

func (f *fakeReports) Insert(_ context.Context, r *domain.Report) error {
	f.inserted = append(f.inserted, r)
	return nil
}
func (f *fakeReports) FindByID(context.Context, bson.ObjectID) (*domain.Report, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeReports) CountRecentByTarget(_ context.Context, targetID bson.ObjectID, _ time.Time) (int, error) {
	count := 0
	for _, r := range f.inserted {
		if r.TargetID == targetID {
			count++
		}
	}
	return count, nil
}
func (f *fakeReports) ListByTarget(context.Context, bson.ObjectID) ([]*domain.Report, error) {
	return nil, nil
}
func (f *fakeReports) List(context.Context, domain.ReportStatus) ([]*domain.Report, error) {
	return nil, nil
}
func (f *fakeReports) Update(_ context.Context, r *domain.Report) error {
	f.updated = append(f.updated, r)
	return nil
}

type fakeListings struct {
	byID map[bson.ObjectID]*domain.Listing
}

func (f *fakeListings) Create(context.Context, *domain.Listing) error { return nil }
func (f *fakeListings) FindByID(_ context.Context, id bson.ObjectID) (*domain.Listing, error) {
	l, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return l, nil
}
func (f *fakeListings) FindByCycleID(context.Context, bson.ObjectID) (*domain.Listing, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeListings) Update(_ context.Context, l *domain.Listing) error {
	f.byID[l.ID] = l
	return nil
}

type fakeClock struct{ now time.Time }

func (f fakeClock) Now() time.Time { return f.now }

func TestCreateReport_Execute_FlagsListingAtThreshold(t *testing.T) {
	listingID := bson.NewObjectID()
	reports := &fakeReports{}
	listings := &fakeListings{byID: map[bson.ObjectID]*domain.Listing{listingID: {ID: listingID}}}
	uc := &CreateReport{Reports: reports, Listings: listings, Clock: fakeClock{now: time.Now()}}

	for range RepeatedReportThreshold - 1 {
		if _, err := uc.Execute(context.Background(), CreateReportInput{
			ReporterID: bson.NewObjectID(), TargetID: listingID, TargetType: domain.ReportTargetListing, Reason: "prix suspect",
		}); err != nil {
			t.Fatalf("Execute() error = %v, want nil", err)
		}
	}
	if listings.byID[listingID].Flagged {
		t.Fatal("Flagged = true before the threshold was reached")
	}

	rp, err := uc.Execute(context.Background(), CreateReportInput{
		ReporterID: bson.NewObjectID(), TargetID: listingID, TargetType: domain.ReportTargetListing, Reason: "paiement hors plateforme",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if rp.Status != domain.ReportUnderReview {
		t.Errorf("report.Status = %v, want %v", rp.Status, domain.ReportUnderReview)
	}
	if !listings.byID[listingID].Flagged {
		t.Error("Flagged = false at the threshold, want true")
	}
}

func TestCreateReport_Execute_RejectsUnsupportedTargetTypeAndEmptyReason(t *testing.T) {
	uc := &CreateReport{Reports: &fakeReports{}, Listings: &fakeListings{byID: map[bson.ObjectID]*domain.Listing{}}, Clock: fakeClock{now: time.Now()}}

	_, err := uc.Execute(context.Background(), CreateReportInput{
		TargetID: bson.NewObjectID(), TargetType: domain.ReportTargetReservation, Reason: "litige",
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("reservation target type: error = %v, want domain.ErrInvalid (that's OpenDispute's job)", err)
	}

	_, err = uc.Execute(context.Background(), CreateReportInput{
		TargetID: bson.NewObjectID(), TargetType: domain.ReportTargetProfile, Reason: "",
	})
	if !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("empty reason: error = %v, want domain.ErrInvalid", err)
	}
}

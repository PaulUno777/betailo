package httpserver

import (
	"net/http"
	"testing"
)

func TestReport_HTTP_ThresholdFlagsListingAndHidesItFromMarketplace(t *testing.T) {
	router := newTestRouter()
	listingID, _ := setupListing(t, router, 120)

	browse := doJSON(t, router, http.MethodGet, "/listings", nil, "")
	found := false
	for _, item := range decodeJSON[marketplaceSearchResponse](t, browse).Items {
		if item.Listing.ID == listingID {
			found = true
		}
	}
	if !found {
		t.Fatal("listing should be visible before any report")
	}

	for i := range 3 {
		reporterToken, _ := registerAndLogin(t, router, uniqueEmail("reporter"), "buyer")
		rec := doJSON(t, router, http.MethodPost, "/reports", createReportRequest{
			TargetID: listingID, TargetType: "listing", Reason: "prix anormalement bas",
		}, reporterToken)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("report #%d status = %d, want %d; body = %s", i+1, rec.Code, http.StatusAccepted, rec.Body.String())
		}
	}

	browseAfter := doJSON(t, router, http.MethodGet, "/listings", nil, "")
	for _, item := range decodeJSON[marketplaceSearchResponse](t, browseAfter).Items {
		if item.Listing.ID == listingID {
			t.Error("flagged listing still appears in the public marketplace")
		}
	}
}

func TestReport_HTTP_RequiresAuth(t *testing.T) {
	router := newTestRouter()
	listingID, _ := setupListing(t, router, 120)

	rec := doJSON(t, router, http.MethodPost, "/reports", createReportRequest{
		TargetID: listingID, TargetType: "listing", Reason: "test",
	}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

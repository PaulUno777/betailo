package httpserver

import (
	"net/http"
	"testing"
	"time"
)

func TestFarmProfile_HTTP_TrustBaselineAndPhonePrivacy(t *testing.T) {
	router := newTestRouter()
	farmerToken, _ := registerAndLogin(t, router, uniqueEmail("farmer"), "farmer")
	createRec := doJSON(t, router, http.MethodPost, "/farms", farmRequest{
		Name: "Ferme des Collines", Location: "Dschang", Description: "Élevage familial", ContactPhone: "6 90 12 34 56",
	}, farmerToken)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create farm status = %d; body = %s", createRec.Code, createRec.Body.String())
	}
	created := decodeJSON[farmResponse](t, createRec)
	if created.ContactPhone != "+237690123456" {
		t.Errorf("owner sees contactPhone = %q, want normalized E.164", created.ContactPhone)
	}

	anon := decodeJSON[farmProfileResponse](t, doJSON(t, router, http.MethodGet, "/farms/"+created.ID, nil, ""))
	if anon.ContactPhone != "" {
		t.Errorf("anonymous caller sees contactPhone = %q, want it withheld", anon.ContactPhone)
	}
	if anon.TrustScore != 50 {
		t.Errorf("trustScore = %v, want the 50 baseline right after farm creation", anon.TrustScore)
	}
	if anon.Description != "Élevage familial" || anon.MemberSince.IsZero() {
		t.Errorf("profile = %+v, want description and memberSince", anon)
	}

	buyerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")
	signedIn := decodeJSON[farmProfileResponse](t, doJSON(t, router, http.MethodGet, "/farms/"+created.ID, nil, buyerToken))
	if signedIn.ContactPhone != "+237690123456" {
		t.Errorf("signed-in buyer sees contactPhone = %q, want it shown", signedIn.ContactPhone)
	}

	bad := doJSON(t, router, http.MethodPost, "/farms", farmRequest{Name: "X", Location: "Y", ContactPhone: "123"}, farmerToken)
	if bad.Code != http.StatusBadRequest {
		t.Errorf("invalid phone status = %d, want 400", bad.Code)
	}
}

func TestHealthEvents_HTTP_AppearInCycleTimeline(t *testing.T) {
	router := newTestRouter()
	listingID, farmerToken := setupListing(t, router, 120)
	cycleID := *decodeJSON[listingResponse](t, doJSON(t, router, http.MethodGet, "/listings/"+listingID, nil, "")).CycleID

	rec := doJSON(t, router, http.MethodPost, "/cycles/"+cycleID+"/health-events", healthEventRequest{
		Title: "Vaccination Marek", Detail: "120 poussins", Date: time.Now().Add(-time.Hour),
	}, farmerToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("add health event status = %d; body = %s", rec.Code, rec.Body.String())
	}

	buyerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")
	forbidden := doJSON(t, router, http.MethodPost, "/cycles/"+cycleID+"/health-events", healthEventRequest{Title: "x", Date: time.Now()}, buyerToken)
	if forbidden.Code != http.StatusForbidden {
		t.Errorf("non-owner status = %d, want 403", forbidden.Code)
	}

	detail := decodeJSON[cycleDetailResponse](t, doJSON(t, router, http.MethodGet, "/cycles/"+cycleID, nil, ""))
	found := false
	for _, s := range detail.Timeline {
		if s.Kind == "health" && s.Title == "Vaccination Marek" && s.Detail == "120 poussins" && s.State == "done" {
			found = true
		}
	}
	if !found {
		t.Fatalf("timeline = %+v, want the health event as a done step", detail.Timeline)
	}
}

func TestFavorites_HTTP_SaveFlagListRemove(t *testing.T) {
	router := newTestRouter()
	listingID, _ := setupListing(t, router, 120)
	buyerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")

	if rec := doJSON(t, router, http.MethodPut, "/listings/"+listingID+"/favorite", nil, buyerToken); rec.Code != http.StatusNoContent {
		t.Fatalf("save status = %d; body = %s", rec.Code, rec.Body.String())
	}
	// Idempotent: a double tap is not an error.
	if rec := doJSON(t, router, http.MethodPut, "/listings/"+listingID+"/favorite", nil, buyerToken); rec.Code != http.StatusNoContent {
		t.Fatalf("second save status = %d, want 204", rec.Code)
	}

	search := decodeJSON[marketplaceSearchResponse](t, doJSON(t, router, http.MethodGet, "/listings", nil, buyerToken))
	if len(search.Items) != 1 || !search.Items[0].Favorited {
		t.Fatalf("signed-in search = %+v, want the listing flagged favorited", search.Items)
	}
	anon := decodeJSON[marketplaceSearchResponse](t, doJSON(t, router, http.MethodGet, "/listings", nil, ""))
	if len(anon.Items) != 1 || anon.Items[0].Favorited {
		t.Fatalf("anonymous search = %+v, want no favorited flag", anon.Items)
	}

	mine := decodeJSON[marketplaceSearchResponse](t, doJSON(t, router, http.MethodGet, "/me/favorites", nil, buyerToken))
	if len(mine.Items) != 1 || mine.Items[0].Listing.ID != listingID {
		t.Fatalf("my favorites = %+v, want the saved listing", mine.Items)
	}

	if rec := doJSON(t, router, http.MethodDelete, "/listings/"+listingID+"/favorite", nil, buyerToken); rec.Code != http.StatusNoContent {
		t.Fatalf("remove status = %d", rec.Code)
	}
	mine = decodeJSON[marketplaceSearchResponse](t, doJSON(t, router, http.MethodGet, "/me/favorites", nil, buyerToken))
	if len(mine.Items) != 0 {
		t.Fatalf("my favorites after removal = %+v, want empty", mine.Items)
	}
}

func TestListingDetail_HTTP_OnePayloadWithViewerFlags(t *testing.T) {
	router := newTestRouter()
	listingID, farmerToken := setupListing(t, router, 120)
	buyerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")

	rec := doJSON(t, router, http.MethodGet, "/listings/"+listingID+"/detail", nil, buyerToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("detail status = %d; body = %s", rec.Code, rec.Body.String())
	}
	d := decodeJSON[listingDetailResponse](t, rec)
	if d.Listing.ID != listingID || d.Cycle.ID == "" || d.Farm.Name != "Ferme des Collines" || len(d.Cycle.Timeline) == 0 {
		t.Fatalf("detail = %+v, want listing, cycle with timeline, and farm profile", d)
	}
	if !d.CanReserve || d.IsOwner {
		t.Errorf("buyer: canReserve=%v isOwner=%v, want true/false", d.CanReserve, d.IsOwner)
	}

	own := decodeJSON[listingDetailResponse](t, doJSON(t, router, http.MethodGet, "/listings/"+listingID+"/detail", nil, farmerToken))
	if own.CanReserve || !own.IsOwner {
		t.Errorf("owner: canReserve=%v isOwner=%v, want false/true (no reserving your own listing)", own.CanReserve, own.IsOwner)
	}

	if rec := doJSON(t, router, http.MethodDelete, "/listings/"+listingID, nil, farmerToken); rec.Code >= 300 {
		t.Fatalf("unpublish status = %d", rec.Code)
	}
	if rec := doJSON(t, router, http.MethodGet, "/listings/"+listingID+"/detail", nil, buyerToken); rec.Code != http.StatusNotFound {
		t.Errorf("unpublished listing, buyer: status = %d, want 404", rec.Code)
	}
	if rec := doJSON(t, router, http.MethodGet, "/listings/"+listingID, nil, ""); rec.Code != http.StatusNotFound {
		t.Errorf("unpublished listing via GET /listings/{id}, anonymous: status = %d, want 404", rec.Code)
	}
	if rec := doJSON(t, router, http.MethodGet, "/listings/"+listingID+"/detail", nil, farmerToken); rec.Code != http.StatusOK {
		t.Errorf("unpublished listing, owner: status = %d, want 200", rec.Code)
	}
}

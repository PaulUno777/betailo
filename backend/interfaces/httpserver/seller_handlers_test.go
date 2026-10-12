package httpserver

import (
	"net/http"
	"testing"
	"time"
)

// newFarmer registers a farmer with a farm and returns their token and
// farm ID.
func newFarmer(t *testing.T, router http.Handler, location string) (token, farmID string) {
	t.Helper()
	token, _ = registerAndLogin(t, router, uniqueEmail("farmer"), "farmer")
	farm := decodeJSON[farmResponse](t, doJSON(t, router, http.MethodPost, "/farms", farmRequest{Name: "Ferme " + location, Location: location}, token))
	return token, farm.ID
}

func quickListing(species string, price float64) quickListingRequest {
	return quickListingRequest{Species: species, Breed: "Brahma", AgeWeeks: 10, Quantity: 40, UnitPrice: price, Title: "Jeunes poulets Brahma — disponibles"}
}

func TestSeller_HTTP_WizardDraftPhotosPublish(t *testing.T) {
	router := newTestRouter()
	token, farmID := newFarmer(t, router, "Dschang")

	rec := doJSON(t, router, http.MethodPost, "/farms/"+farmID+"/quick-listings", quickListing("jeunes-poulets", 12000), token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("quick create status = %d; body = %s", rec.Code, rec.Body.String())
	}
	out := decodeJSON[quickListingResponse](t, rec)
	l, c := out.Listing, out.Cycle
	if l.Status != "draft" || l.Unit != "sujet" || l.Breed != "Brahma" || c.Type != "engraissement" || c.Status != "in_progress" || c.Reference == "" {
		t.Fatalf("created = %+v / %+v, want a draft priced per sujet on an in-progress engraissement cycle with a reference", l, c)
	}

	// A draft isn't for sale yet.
	if search := decodeJSON[marketplaceSearchResponse](t, doJSON(t, router, http.MethodGet, "/listings", nil, "")); search.Total != 0 {
		t.Fatalf("draft showed up in the marketplace: %+v", search.Items)
	}
	if pub := doJSON(t, router, http.MethodPost, "/listings/"+l.ID+"/publish", nil, token); pub.Code != http.StatusBadRequest {
		t.Fatalf("publish without photos status = %d, want 400", pub.Code)
	} else if code := decodeJSON[errorResponse](t, pub).Code; code != "listing.photo_required" {
		// Clients translate this code; the English message is never shown.
		t.Fatalf("publish without photos code = %q, want listing.photo_required", code)
	}

	key := decodeJSON[presignPhotoResponse](t, doJSON(t, router, http.MethodPost, "/listings/"+l.ID+"/photos/presign", nil, token)).Key
	if set := doJSON(t, router, http.MethodPut, "/listings/"+l.ID+"/photos", setListingPhotosRequest{Keys: []string{key}}, token); set.Code != http.StatusOK {
		t.Fatalf("set one photo status = %d; body = %s", set.Code, set.Body.String())
	}
	if pub := doJSON(t, router, http.MethodPost, "/listings/"+l.ID+"/publish", nil, token); pub.Code != http.StatusOK || decodeJSON[listingResponse](t, pub).Status != "published" {
		t.Fatalf("publish status = %d; body = %s", pub.Code, pub.Body.String())
	}
	if search := decodeJSON[marketplaceSearchResponse](t, doJSON(t, router, http.MethodGet, "/listings", nil, "")); search.Total != 1 {
		t.Fatalf("published listing missing from the marketplace")
	}

	mine := decodeJSON[[]farmListingResponse](t, doJSON(t, router, http.MethodGet, "/farms/"+farmID+"/listings", nil, token))
	if len(mine) != 1 || mine[0].Listing.ID != l.ID || mine[0].Cycle.ID != c.ID || len(mine[0].PhotoKeys) != 1 || mine[0].PhotoKeys[0] != key {
		t.Fatalf("farm listings = %+v, want the one listing with its cycle", mine)
	}
}

func TestSeller_HTTP_WizardValidationAndRoles(t *testing.T) {
	router := newTestRouter()
	token, farmID := newFarmer(t, router, "Dschang")
	buyer, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")

	if rec := doJSON(t, router, http.MethodPost, "/farms/"+farmID+"/quick-listings", quickListing("jeunes-poulets", 12000), buyer); rec.Code != http.StatusForbidden {
		t.Errorf("buyer quick-create status = %d, want 403", rec.Code)
	}
	if rec := doJSON(t, router, http.MethodPost, "/farms/"+farmID+"/quick-listings", quickListing("licornes", 12000), token); rec.Code != http.StatusBadRequest {
		t.Errorf("unknown species status = %d, want 400", rec.Code)
	}
	past := time.Now().Add(-72 * time.Hour)
	req := quickListing("porcs", 50000)
	req.AvailableFrom = &past
	if rec := doJSON(t, router, http.MethodPost, "/farms/"+farmID+"/quick-listings", req, token); rec.Code != http.StatusBadRequest {
		t.Errorf("past availability status = %d, want 400", rec.Code)
	}
	future := time.Now().Add(10 * 24 * time.Hour)
	req.AvailableFrom = &future
	planned := decodeJSON[quickListingResponse](t, doJSON(t, router, http.MethodPost, "/farms/"+farmID+"/quick-listings", req, token))
	if planned.Cycle.Status != "planned" || planned.Listing.Unit != "porcelet" {
		t.Errorf("future availability = %+v, want a planned cycle priced per porcelet", planned)
	}
}

func TestSeller_HTTP_PriceHint(t *testing.T) {
	router := newTestRouter()
	publish := func(token, farmID string, price float64) {
		l := decodeJSON[quickListingResponse](t, doJSON(t, router, http.MethodPost, "/farms/"+farmID+"/quick-listings", quickListing("jeunes-poulets", price), token)).Listing
		key := decodeJSON[presignPhotoResponse](t, doJSON(t, router, http.MethodPost, "/listings/"+l.ID+"/photos/presign", nil, token)).Key
		doJSON(t, router, http.MethodPut, "/listings/"+l.ID+"/photos", setListingPhotosRequest{Keys: []string{key}}, token)
		doJSON(t, router, http.MethodPost, "/listings/"+l.ID+"/publish", nil, token)
	}
	west, westFarm := newFarmer(t, router, "Dschang")
	publish(west, westFarm, 10000)
	publish(west, westFarm, 12000)

	if rec := doJSON(t, router, http.MethodGet, "/listings/price-hint?species=jeunes-poulets&region=Dschang", nil, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("2 samples: status = %d, want 204", rec.Code)
	}

	littoral, litFarm := newFarmer(t, router, "Douala")
	publish(littoral, litFarm, 14000)
	hint := decodeJSON[priceHintResponse](t, doJSON(t, router, http.MethodGet, "/listings/price-hint?species=jeunes-poulets&region=Dschang", nil, ""))
	if hint.Count != 3 || hint.Regional || hint.Low != 11000 || hint.Median != 12000 || hint.High != 13000 {
		t.Fatalf("hint = %+v, want platform-wide 11000..13000 around 12000 (only 2 in Dschang)", hint)
	}
}

func TestSeller_HTTP_CycleEditCantUnsellReservedPlaces(t *testing.T) {
	router := newTestRouter()
	listingID, farmerToken := setupListing(t, router, 120)
	buyer, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")
	doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{Quantity: 100, PayNow: true, PaymentMethod: "card"}, buyer)
	cycleID := *decodeJSON[listingResponse](t, doJSON(t, router, http.MethodGet, "/listings/"+listingID, nil, "")).CycleID

	starts := time.Now().Add(48 * time.Hour)
	shrink := updateCycleRequest{StartsAt: starts, Capacity: 90}
	if rec := doJSON(t, router, http.MethodPut, "/cycles/"+cycleID, shrink, farmerToken); rec.Code != http.StatusConflict {
		t.Fatalf("shrinking below the 100 reserved: status = %d, want 409", rec.Code)
	}
	grow := updateCycleRequest{StartsAt: starts, Capacity: 150, MinPerReservation: 5, MaxPerReservation: 40}
	rec := doJSON(t, router, http.MethodPut, "/cycles/"+cycleID, grow, farmerToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("grow status = %d; body = %s", rec.Code, rec.Body.String())
	}
	if c := decodeJSON[cycleResponse](t, rec); c.Capacity != 150 || c.AvailableQuantity != 50 || c.MaxPerReservation != 40 {
		t.Fatalf("after growing = %+v, want capacity 150, 50 still available, max 40", c)
	}
	if rec := doJSON(t, router, http.MethodPut, "/cycles/"+cycleID, grow, buyer); rec.Code != http.StatusForbidden {
		t.Errorf("buyer edit status = %d, want 403", rec.Code)
	}
}

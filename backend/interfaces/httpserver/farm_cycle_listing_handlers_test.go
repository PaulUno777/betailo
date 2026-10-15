package httpserver

import (
	"net/http"
	"testing"
	"time"
)

// TestDemoFlow_HTTP_FarmCycleListing exercises the roadmap's own
// suggested demo path end to end over real HTTP: a farmer creates a
// farm, then a cycle on it, then publishes a listing for that cycle —
// and a buyer can browse it without authenticating. Every status code
// and every field returned is real request/response traffic through
// the actual router, not a direct usecase call.
func TestDemoFlow_HTTP_FarmCycleListing(t *testing.T) {
	router := newTestRouter()
	farmerToken, _ := registerAndLogin(t, router, uniqueEmail("farmer"), "farmer")

	farmRec := doJSON(t, router, http.MethodPost, "/farms", farmRequest{
		Name: "Ferme du Soleil", Location: "Douala", Species: []string{"poussins"},
	}, farmerToken)
	if farmRec.Code != http.StatusCreated {
		t.Fatalf("create farm status = %d, want %d; body = %s", farmRec.Code, http.StatusCreated, farmRec.Body.String())
	}
	farmOut := decodeJSON[farmResponse](t, farmRec)

	starts := time.Now().Add(24 * time.Hour)
	cycleRec := doJSON(t, router, http.MethodPost, "/cycles", createCycleRequest{
		FarmID: farmOut.ID, Species: "poussins", Type: "ponte", Capacity: 50, StartsAt: starts,
	}, farmerToken)
	if cycleRec.Code != http.StatusCreated {
		t.Fatalf("create cycle status = %d, want %d; body = %s", cycleRec.Code, http.StatusCreated, cycleRec.Body.String())
	}
	cycleOut := decodeJSON[cycleResponse](t, cycleRec)
	if cycleOut.AvailableQuantity != 50 {
		t.Errorf("AvailableQuantity = %d, want 50 (equal to Capacity on creation)", cycleOut.AvailableQuantity)
	}

	listingRec := doJSON(t, router, http.MethodPost, "/listings", createListingRequest{
		CycleID: cycleOut.ID, Title: "Lot de poules pondeuses", UnitPrice: 15.5,
	}, farmerToken)
	if listingRec.Code != http.StatusCreated {
		t.Fatalf("create listing status = %d, want %d; body = %s", listingRec.Code, http.StatusCreated, listingRec.Body.String())
	}
	listingOut := decodeJSON[listingResponse](t, listingRec)
	if listingOut.CycleID == nil || *listingOut.CycleID != cycleOut.ID {
		t.Errorf("listing.CycleID = %v, want %v", listingOut.CycleID, cycleOut.ID)
	}
	if listingOut.Status != "published" {
		t.Errorf("listing.Status = %q, want %q", listingOut.Status, "published")
	}

	// A buyer (no token at all) can browse the marketplace.
	browseRec := doJSON(t, router, http.MethodGet, "/listings", nil, "")
	if browseRec.Code != http.StatusOK {
		t.Fatalf("browse listings status = %d, want %d", browseRec.Code, http.StatusOK)
	}
	published := decodeJSON[marketplaceSearchResponse](t, browseRec)
	found := false
	for _, item := range published.Items {
		if item.Listing.ID == listingOut.ID {
			found = true
		}
	}
	if !found {
		t.Error("the published listing doesn't appear in the public browse list")
	}

	getFarmRec := doJSON(t, router, http.MethodGet, "/farms/"+farmOut.ID, nil, "")
	if getFarmRec.Code != http.StatusOK {
		t.Fatalf("GET /farms/{id} status = %d, want %d (public read)", getFarmRec.Code, http.StatusOK)
	}

	cyclesRec := doJSON(t, router, http.MethodGet, "/farms/"+farmOut.ID+"/cycles", nil, "")
	if cyclesRec.Code != http.StatusOK {
		t.Fatalf("GET /farms/{farmId}/cycles status = %d, want %d", cyclesRec.Code, http.StatusOK)
	}
	if got := decodeJSON[[]cycleResponse](t, cyclesRec); len(got) != 1 {
		t.Fatalf("got %d cycles for the farm, want 1", len(got))
	}
}

// There is one kind of account: someone who only ever bought can open
// a farm, and GET /me then lists it.
func TestCreateFarm_HTTP_AnyAccountCanOpenOne(t *testing.T) {
	router := newTestRouter()
	token, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")

	rec := doJSON(t, router, http.MethodPost, "/farms", farmRequest{Name: "Ferme", Location: "Douala"}, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	me := decodeJSON[meResponse](t, doJSON(t, router, http.MethodGet, "/me", nil, token))
	if len(me.Farms) != 1 || me.Farms[0].Name != "Ferme" || me.Farms[0].Review == nil {
		t.Errorf("GET /me farms = %+v, want the new farm with its review state", me.Farms)
	}
}

func TestCreateFarm_HTTP_NoToken(t *testing.T) {
	router := newTestRouter()
	rec := doJSON(t, router, http.MethodPost, "/farms", farmRequest{Name: "Ferme", Location: "Douala"}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestUpdateFarm_HTTP_NotOwnerForbidden(t *testing.T) {
	router := newTestRouter()
	ownerToken, _ := registerAndLogin(t, router, uniqueEmail("owner"), "farmer")
	otherToken, _ := registerAndLogin(t, router, uniqueEmail("other"), "farmer")

	farmRec := doJSON(t, router, http.MethodPost, "/farms", farmRequest{Name: "Ferme", Location: "Douala"}, ownerToken)
	farmOut := decodeJSON[farmResponse](t, farmRec)

	rec := doJSON(t, router, http.MethodPut, "/farms/"+farmOut.ID, farmRequest{Name: "Hijacked", Location: "Douala"}, otherToken)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d — a different farmer must not update someone else's farm", rec.Code, http.StatusForbidden)
	}
}

func TestCreateCycle_HTTP_Validation(t *testing.T) {
	router := newTestRouter()
	token, _ := registerAndLogin(t, router, uniqueEmail("farmer"), "farmer")
	farmRec := doJSON(t, router, http.MethodPost, "/farms", farmRequest{Name: "Ferme", Location: "Douala"}, token)
	farmOut := decodeJSON[farmResponse](t, farmRec)

	rec := doJSON(t, router, http.MethodPost, "/cycles", createCycleRequest{
		FarmID: farmOut.ID, Species: "poussins", Type: "ponte", Capacity: 0, StartsAt: time.Now(),
	}, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d — zero capacity must be rejected", rec.Code, http.StatusBadRequest)
	}
}

func TestUpdateCycleStatus_HTTP_InvalidTransition(t *testing.T) {
	router := newTestRouter()
	token, _ := registerAndLogin(t, router, uniqueEmail("farmer"), "farmer")
	farmRec := doJSON(t, router, http.MethodPost, "/farms", farmRequest{Name: "Ferme", Location: "Douala"}, token)
	farmOut := decodeJSON[farmResponse](t, farmRec)
	cycleRec := doJSON(t, router, http.MethodPost, "/cycles", createCycleRequest{
		FarmID: farmOut.ID, Species: "poussins", Type: "ponte", Capacity: 10, StartsAt: time.Now(),
	}, token)
	cycleOut := decodeJSON[cycleResponse](t, cycleRec)

	// planned -> done directly is not an allowed transition.
	rec := doJSON(t, router, http.MethodPatch, "/cycles/"+cycleOut.ID+"/status", updateCycleStatusRequest{Status: "done"}, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDeleteFarm_HTTP_ConflictWithExistingCycle(t *testing.T) {
	router := newTestRouter()
	token, _ := registerAndLogin(t, router, uniqueEmail("farmer"), "farmer")
	farmRec := doJSON(t, router, http.MethodPost, "/farms", farmRequest{Name: "Ferme", Location: "Douala"}, token)
	farmOut := decodeJSON[farmResponse](t, farmRec)
	doJSON(t, router, http.MethodPost, "/cycles", createCycleRequest{
		FarmID: farmOut.ID, Species: "poussins", Type: "ponte", Capacity: 10, StartsAt: time.Now(),
	}, token)

	rec := doJSON(t, router, http.MethodDelete, "/farms/"+farmOut.ID, nil, token)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d — a farm with an existing cycle must not be deletable", rec.Code, http.StatusConflict)
	}
}

func TestUnpublishListing_HTTP_RemovesFromPublicBrowse(t *testing.T) {
	router := newTestRouter()
	token, _ := registerAndLogin(t, router, uniqueEmail("farmer"), "farmer")
	farmRec := doJSON(t, router, http.MethodPost, "/farms", farmRequest{Name: "Ferme", Location: "Douala"}, token)
	farmOut := decodeJSON[farmResponse](t, farmRec)
	cycleRec := doJSON(t, router, http.MethodPost, "/cycles", createCycleRequest{
		FarmID: farmOut.ID, Species: "poussins", Type: "ponte", Capacity: 10, StartsAt: time.Now(),
	}, token)
	cycleOut := decodeJSON[cycleResponse](t, cycleRec)
	listingRec := doJSON(t, router, http.MethodPost, "/listings", createListingRequest{
		CycleID: cycleOut.ID, Title: "Lot", UnitPrice: 10,
	}, token)
	listingOut := decodeJSON[listingResponse](t, listingRec)

	unpubRec := doJSON(t, router, http.MethodDelete, "/listings/"+listingOut.ID, nil, token)
	if unpubRec.Code != http.StatusNoContent {
		t.Fatalf("unpublish status = %d, want %d", unpubRec.Code, http.StatusNoContent)
	}

	browseRec := doJSON(t, router, http.MethodGet, "/listings", nil, "")
	for _, item := range decodeJSON[marketplaceSearchResponse](t, browseRec).Items {
		if item.Listing.ID == listingOut.ID {
			t.Error("unpublished listing still appears in the public browse list")
		}
	}
}

func TestFarm_HTTP_CountryAndRegion(t *testing.T) {
	router := newTestRouter()
	token, _ := registerAndLogin(t, router, uniqueEmail("farmer"), "farmer")

	rec := doJSON(t, router, http.MethodPost, "/farms", farmRequest{Name: "Ferme", Location: "Dschang", Region: "ouest"}, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d; body = %s", rec.Code, rec.Body.String())
	}
	if f := decodeJSON[farmResponse](t, rec); f.Country != "CM" || f.Region != "ouest" || f.Location != "dschang" {
		t.Errorf("farm = %s/%s/%s, want the default country and the catalog's keys CM/ouest/dschang", f.Country, f.Region, f.Location)
	}

	bad := doJSON(t, router, http.MethodPost, "/farms", farmRequest{Name: "Ferme", Location: "Douala", Region: "Atlantique"}, token)
	if bad.Code != http.StatusBadRequest || decodeJSON[errorResponse](t, bad).Code != "farm.region_invalid" {
		t.Errorf("unknown Cameroonian region: status = %d, body = %s; want 400 farm.region_invalid", bad.Code, bad.Body.String())
	}

	// Any other country takes its region as typed, and the contact
	// phone is read as a national number of that country.
	abroad := doJSON(t, router, http.MethodPost, "/farms", farmRequest{Name: "Ferme", Location: "Libreville", Country: "ga", Region: "Estuaire", ContactPhone: "06 12 34 56"}, token)
	if f := decodeJSON[farmResponse](t, abroad); abroad.Code != http.StatusCreated || f.Country != "GA" || f.Region != "Estuaire" || f.ContactPhone != "+2416123456" {
		t.Errorf("farm abroad: status = %d, farm = %+v", abroad.Code, f)
	}

}

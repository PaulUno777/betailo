package httpserver

import (
	"net/http"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestListMyFarms_HTTP(t *testing.T) {
	router := newTestRouter()
	token, _ := registerAndLogin(t, router, uniqueEmail("farmer"), "farmer")
	doJSON(t, router, http.MethodPost, "/farms", farmRequest{Name: "A", Location: "Douala"}, token)
	doJSON(t, router, http.MethodPost, "/farms", farmRequest{Name: "B", Location: "Yaoundé"}, token)

	rec := doJSON(t, router, http.MethodGet, "/farms/mine", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := decodeJSON[[]farmResponse](t, rec); len(got) != 2 {
		t.Fatalf("got %d farms, want 2", len(got))
	}
}

func TestGetListing_HTTP_NotFound(t *testing.T) {
	router := newTestRouter()
	rec := doJSON(t, router, http.MethodGet, "/listings/"+bson.NewObjectID().Hex(), nil, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestUpdateListing_HTTP_NotOwnerForbidden(t *testing.T) {
	router := newTestRouter()
	ownerToken, _ := registerAndLogin(t, router, uniqueEmail("owner"), "farmer")
	otherToken, _ := registerAndLogin(t, router, uniqueEmail("other"), "farmer")

	farmRec := doJSON(t, router, http.MethodPost, "/farms", farmRequest{Name: "Ferme", Location: "Douala"}, ownerToken)
	farmOut := decodeJSON[farmResponse](t, farmRec)
	cycleRec := doJSON(t, router, http.MethodPost, "/cycles", createCycleRequest{
		FarmID: farmOut.ID, Species: "poussins", Type: "ponte", Capacity: 10, StartsAt: time.Now(),
	}, ownerToken)
	cycleOut := decodeJSON[cycleResponse](t, cycleRec)
	listingRec := doJSON(t, router, http.MethodPost, "/listings", createListingRequest{
		CycleID: cycleOut.ID, Title: "Lot", UnitPrice: 10,
	}, ownerToken)
	listingOut := decodeJSON[listingResponse](t, listingRec)

	rec := doJSON(t, router, http.MethodPut, "/listings/"+listingOut.ID, createListingRequest{
		CycleID: cycleOut.ID, Title: "Hijacked", UnitPrice: 999,
	}, otherToken)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestDeleteCycle_HTTP_SuccessAndConflict(t *testing.T) {
	router := newTestRouter()
	token, _ := registerAndLogin(t, router, uniqueEmail("farmer"), "farmer")
	farmRec := doJSON(t, router, http.MethodPost, "/farms", farmRequest{Name: "Ferme", Location: "Douala"}, token)
	farmOut := decodeJSON[farmResponse](t, farmRec)
	cycleRec := doJSON(t, router, http.MethodPost, "/cycles", createCycleRequest{
		FarmID: farmOut.ID, Species: "poussins", Type: "ponte", Capacity: 10, StartsAt: time.Now(),
	}, token)
	cycleOut := decodeJSON[cycleResponse](t, cycleRec)

	// No listing yet: delete succeeds.
	delRec := doJSON(t, router, http.MethodDelete, "/cycles/"+cycleOut.ID, nil, token)
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("delete (no listing) status = %d, want %d", delRec.Code, http.StatusNoContent)
	}

	// A second cycle, now with a listing: delete is blocked.
	cycle2Rec := doJSON(t, router, http.MethodPost, "/cycles", createCycleRequest{
		FarmID: farmOut.ID, Species: "poussins", Type: "ponte", Capacity: 10, StartsAt: time.Now(),
	}, token)
	cycle2Out := decodeJSON[cycleResponse](t, cycle2Rec)
	doJSON(t, router, http.MethodPost, "/listings", createListingRequest{
		CycleID: cycle2Out.ID, Title: "Lot", UnitPrice: 10,
	}, token)

	conflictRec := doJSON(t, router, http.MethodDelete, "/cycles/"+cycle2Out.ID, nil, token)
	if conflictRec.Code != http.StatusConflict {
		t.Fatalf("delete (has listing) status = %d, want %d", conflictRec.Code, http.StatusConflict)
	}
}

func TestCreateListing_HTTP_UnknownCycle(t *testing.T) {
	router := newTestRouter()
	token, _ := registerAndLogin(t, router, uniqueEmail("farmer"), "farmer")

	rec := doJSON(t, router, http.MethodPost, "/listings", createListingRequest{
		CycleID: bson.NewObjectID().Hex(), Title: "Lot", UnitPrice: 10,
	}, token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

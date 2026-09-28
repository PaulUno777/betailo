package httpserver

import (
	"net/http"
	"testing"
	"time"
)

func TestListingPhotos_HTTP_PresignThenSetThenResolvedOnRead(t *testing.T) {
	router := newTestRouter()
	listingID, farmerToken := setupListing(t, router, 120)

	var keys []string
	for range 3 {
		rec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/photos/presign", nil, farmerToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("presign status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		out := decodeJSON[presignPhotoResponse](t, rec)
		if out.Key == "" || out.UploadURL == "" {
			t.Fatalf("presign response = %+v, want both fields set", out)
		}
		keys = append(keys, out.Key)
	}

	// Fewer than the minimum must be rejected.
	tooFew := doJSON(t, router, http.MethodPut, "/listings/"+listingID+"/photos", setListingPhotosRequest{Keys: keys[:1]}, farmerToken)
	if tooFew.Code != http.StatusBadRequest {
		t.Fatalf("too-few-photos status = %d, want %d", tooFew.Code, http.StatusBadRequest)
	}

	setRec := doJSON(t, router, http.MethodPut, "/listings/"+listingID+"/photos", setListingPhotosRequest{Keys: keys}, farmerToken)
	if setRec.Code != http.StatusOK {
		t.Fatalf("set photos status = %d, want %d; body = %s", setRec.Code, http.StatusOK, setRec.Body.String())
	}

	getRec := doJSON(t, router, http.MethodGet, "/listings/"+listingID, nil, "")
	got := decodeJSON[listingResponse](t, getRec)
	if len(got.Photos) != 3 {
		t.Fatalf("Photos = %d, want 3", len(got.Photos))
	}
	for _, url := range got.Photos {
		if url == "" {
			t.Error("a photo resolved to an empty URL")
		}
	}
}

func TestSearchListings_HTTP_FiltersBySpeciesAndPrice(t *testing.T) {
	router := newTestRouter()
	farmerToken, _ := registerAndLogin(t, router, uniqueEmail("farmer"), "farmer")

	farmRec := doJSON(t, router, http.MethodPost, "/farms", farmRequest{Name: "Ferme A", Location: "Douala"}, farmerToken)
	farmID := decodeJSON[farmResponse](t, farmRec).ID

	makeListing := func(species string, price float64) string {
		cycleRec := doJSON(t, router, http.MethodPost, "/cycles", createCycleRequest{
			FarmID: farmID, Species: species, Type: "engraissement", Capacity: 50, StartsAt: time.Now().Add(24 * time.Hour),
		}, farmerToken)
		cycleID := decodeJSON[cycleResponse](t, cycleRec).ID
		listingRec := doJSON(t, router, http.MethodPost, "/listings", createListingRequest{
			CycleID: cycleID, Title: species, UnitPrice: price,
		}, farmerToken)
		return decodeJSON[listingResponse](t, listingRec).ID
	}

	cheapChicken := makeListing("poulet", 1000)
	expensivePig := makeListing("porc", 20000)

	rec := doJSON(t, router, http.MethodGet, "/listings?species=poulet&maxPrice=5000", nil, "")
	items := decodeJSON[marketplaceSearchResponse](t, rec).Items
	foundCheap, foundExpensive := false, false
	for _, item := range items {
		if item.Listing.ID == cheapChicken {
			foundCheap = true
		}
		if item.Listing.ID == expensivePig {
			foundExpensive = true
		}
	}
	if !foundCheap {
		t.Error("species=poulet&maxPrice=5000 should include the cheap chicken listing")
	}
	if foundExpensive {
		t.Error("species=poulet&maxPrice=5000 should exclude the pig listing")
	}
}

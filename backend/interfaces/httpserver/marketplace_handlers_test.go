package httpserver

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pauluno777/betailo/backend/domain"
)

func TestListingPhotos_HTTP_PresignThenSetThenResolvedOnRead(t *testing.T) {
	router := newTestRouter()
	listingID, farmerToken := setupListing(t, router, 120)

	var keys []string
	for range 3 {
		rec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/photos/presign", photoSizes, farmerToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("presign status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		out := decodeJSON[presignPhotoResponse](t, rec)
		if out.Key == "" || out.UploadURL == "" || out.ThumbUploadURL == "" {
			t.Fatalf("presign response = %+v, want both fields set", out)
		}
		keys = append(keys, out.Key)
	}

	// Fewer than the minimum (one) must be rejected.
	tooFew := doJSON(t, router, http.MethodPut, "/listings/"+listingID+"/photos", setListingPhotosRequest{Keys: []string{}}, farmerToken)
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
	if len(got.Thumbs) != 3 || !strings.HasSuffix(got.Thumbs[0], ".thumb.jpg") {
		t.Fatalf("Thumbs = %v, want one small copy per photo", got.Thumbs)
	}
}

// photoSizes is what the browser declares for a resized photo and its
// small copy.
var photoSizes = presignPhotoRequest{Size: 200 << 10, ThumbSize: 40 << 10}

func TestListingPhotos_HTTP_PresignRefusesUndeclaredOrHeavySizes(t *testing.T) {
	router := newTestRouter()
	listingID, farmerToken := setupListing(t, router, 120)

	for name, body := range map[string]presignPhotoRequest{
		"no size":         {},
		"photo too heavy": {Size: domain.MaxPhotoBytes + 1, ThumbSize: 40 << 10},
		"thumb too heavy": {Size: 200 << 10, ThumbSize: domain.MaxThumbBytes + 1},
	} {
		rec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/photos/presign", body, farmerToken)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d; body = %s", name, rec.Code, http.StatusBadRequest, rec.Body.String())
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

	cheapChicken := makeListing("poussins", 1000)
	expensivePig := makeListing("porcs", 20000)

	rec := doJSON(t, router, http.MethodGet, "/listings?species=poussins&maxPrice=5000", nil, "")
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
		t.Error("species=poussins&maxPrice=5000 should include the cheap chicken listing")
	}
	if foundExpensive {
		t.Error("species=poussins&maxPrice=5000 should exclude the pig listing")
	}

	// A category stands for its sub-categories: the pig, not the chicks.
	rec = doJSON(t, router, http.MethodGet, "/listings?category=porcins", nil, "")
	ids := []string{}
	for _, item := range decodeJSON[marketplaceSearchResponse](t, rec).Items {
		ids = append(ids, item.Listing.ID)
	}
	if !slices.Contains(ids, expensivePig) || slices.Contains(ids, cheapChicken) {
		t.Errorf("category=porcins = %v, want the pig listing only", ids)
	}
}

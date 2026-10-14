package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// uploadFarmProof asks for n proof upload slots and returns their keys
// (no bytes move in these tests; see fakeObjectStorage).
func uploadFarmProof(t *testing.T, router http.Handler, farmID, token string, n int) []string {
	t.Helper()
	keys := make([]string, 0, n)
	for range n {
		rec := doJSON(t, router, http.MethodPost, "/farms/"+farmID+"/verification/presign", nil, token)
		if rec.Code != http.StatusOK {
			t.Fatalf("presign status = %d; body = %s", rec.Code, rec.Body.String())
		}
		keys = append(keys, decodeJSON[farmProofUploadResponse](t, rec).Key)
	}
	return keys
}

// newDraftFarmWithListing opens a farm that still awaits review, with a
// cycle and one listing ready to publish (it has a photo).
func newDraftFarmWithListing(t *testing.T, router http.Handler, tf *testFakes) (token, farmID, listingID string) {
	t.Helper()
	tf.farms.awaitReview = true
	token, _ = registerAndLogin(t, router, uniqueEmail("seller"), "farmer")
	farmRec := doJSON(t, router, http.MethodPost, "/farms", farmRequest{Name: "Ferme Etoga", Location: "Douala"}, token)
	if farmRec.Code != http.StatusCreated {
		t.Fatalf("create farm status = %d; body = %s", farmRec.Code, farmRec.Body.String())
	}
	farm := decodeJSON[farmResponse](t, farmRec)
	if farm.Status != "draft" || farm.Review == nil || !farm.Review.CanSubmit {
		t.Fatalf("new farm = %+v (review %+v), want a draft its owner can submit", farm, farm.Review)
	}
	quick := doJSON(t, router, http.MethodPost, "/farms/"+farm.ID+"/quick-listings", quickListingRequest{Species: "poussins", Quantity: 20, UnitPrice: 700, Title: "Poussins"}, token)
	if quick.Code != http.StatusCreated {
		t.Fatalf("quick listing status = %d; body = %s", quick.Code, quick.Body.String())
	}
	listingID = decodeJSON[quickListingResponse](t, quick).Listing.ID
	up := decodeJSON[presignPhotoResponse](t, doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/photos/presign", nil, token))
	if rec := doJSON(t, router, http.MethodPut, "/listings/"+listingID+"/photos", setListingPhotosRequest{Keys: []string{up.Key}}, token); rec.Code != http.StatusOK {
		t.Fatalf("set photos status = %d; body = %s", rec.Code, rec.Body.String())
	}
	return token, farm.ID, listingID
}

// The whole point of the model: anyone opens a farm and prepares it,
// but it sells only once staff approved the proof its owner submitted.
func TestFarm_HTTP_SellsOnlyOnceStaffApprovedIt(t *testing.T) {
	router, tf := newTestRouterAndFakes()
	token, farmID, listingID := newDraftFarmWithListing(t, router, tf)
	root := adminToken(t, router, tf)

	publish := func() *httptest.ResponseRecorder {
		return doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/publish", nil, token)
	}
	if rec := publish(); rec.Code != http.StatusForbidden || decodeJSON[errorResponse](t, rec).Code != "farm.not_approved" {
		t.Fatalf("publish from a draft farm = %d %s, want 403 farm.not_approved", rec.Code, rec.Body.String())
	}

	// Proof is required, and only keys this farm's own endpoint issued.
	submit := func(body submitFarmRequest) *httptest.ResponseRecorder {
		return doJSON(t, router, http.MethodPost, "/farms/"+farmID+"/verification/submit", body, token)
	}
	if rec := submit(submitFarmRequest{}); rec.Code != http.StatusBadRequest || decodeJSON[errorResponse](t, rec).Code != "farm.id_document_required" {
		t.Fatalf("submit without proof = %d %s, want 400 farm.id_document_required", rec.Code, rec.Body.String())
	}
	keys := uploadFarmProof(t, router, farmID, token, 2)
	if rec := submit(submitFarmRequest{IDDocumentKey: keys[0]}); rec.Code != http.StatusBadRequest {
		t.Fatalf("submit without farm photos = %d, want 400", rec.Code)
	}
	if rec := submit(submitFarmRequest{IDDocumentKey: "listings/someone-else.jpg", PhotoKeys: keys[1:]}); rec.Code != http.StatusBadRequest {
		t.Fatalf("submit with a foreign key = %d, want 400", rec.Code)
	}
	rec := submit(submitFarmRequest{IDDocumentKey: keys[0], PhotoKeys: keys[1:]})
	if rec.Code != http.StatusOK || decodeJSON[farmResponse](t, rec).Status != "pending" {
		t.Fatalf("submit = %d %s, want 200 pending", rec.Code, rec.Body.String())
	}
	if rec := submit(submitFarmRequest{IDDocumentKey: keys[0], PhotoKeys: keys[1:]}); rec.Code != http.StatusConflict {
		t.Errorf("second submit while pending = %d, want 409", rec.Code)
	}
	if rec := publish(); rec.Code != http.StatusForbidden {
		t.Fatalf("publish from a pending farm = %d, want 403", rec.Code)
	}

	// Staff see it in the queue, with the proof behind signed links.
	queue := decodeJSON[[]adminFarmResponse](t, doJSON(t, router, http.MethodGet, "/admin/farms?status=pending", nil, root))
	if len(queue) != 1 || queue[0].ID != farmID || queue[0].Owner == nil {
		t.Fatalf("pending queue = %+v, want the submitted farm with its owner", queue)
	}
	detail := decodeJSON[adminFarmDetailResponse](t, doJSON(t, router, http.MethodGet, "/admin/farms/"+farmID, nil, root))
	if !strings.Contains(detail.IDDocumentURL, keys[0]) || len(detail.PhotoURLs) != 1 {
		t.Fatalf("detail proof = %q %v, want download links for what was submitted", detail.IDDocumentURL, detail.PhotoURLs)
	}
	// The proof never shows on the public profile.
	if body := doJSON(t, router, http.MethodGet, "/farms/"+farmID, nil, "").Body.String(); strings.Contains(body, "farm-verifications") || strings.Contains(body, `"review":`) {
		t.Errorf("public farm profile leaks the review: %s", body)
	}
	if rec := doJSON(t, router, http.MethodGet, "/admin/farms/"+farmID, nil, token); rec.Code != http.StatusForbidden {
		t.Errorf("owner reading the staff view = %d, want 403", rec.Code)
	}

	// Rejected with a reason, the owner resubmits, staff approve.
	review := func(body reviewFarmRequest) *httptest.ResponseRecorder {
		return doJSON(t, router, http.MethodPost, "/admin/farms/"+farmID+"/review", body, root)
	}
	if rec := review(reviewFarmRequest{Decision: "reject"}); rec.Code != http.StatusBadRequest {
		t.Fatalf("reject without a reason = %d, want 400", rec.Code)
	}
	rejected := decodeJSON[farmResponse](t, review(reviewFarmRequest{Decision: "reject", Reason: "Document illisible"}))
	if rejected.Status != "rejected" || rejected.Review.RejectionReason != "Document illisible" || !rejected.Review.CanSubmit {
		t.Fatalf("after reject = %+v (review %+v)", rejected, rejected.Review)
	}
	if rec := submit(submitFarmRequest{IDDocumentKey: keys[0], PhotoKeys: keys[1:]}); rec.Code != http.StatusOK {
		t.Fatalf("resubmit = %d; body = %s", rec.Code, rec.Body.String())
	}
	if rec := review(reviewFarmRequest{Decision: "approve"}); rec.Code != http.StatusOK || decodeJSON[farmResponse](t, rec).Status != "approved" {
		t.Fatalf("approve = %d %s", rec.Code, rec.Body.String())
	}

	if rec := publish(); rec.Code != http.StatusOK {
		t.Fatalf("publish from an approved farm = %d; body = %s", rec.Code, rec.Body.String())
	}
	onMarket := func() bool {
		res := decodeJSON[marketplaceSearchResponse](t, doJSON(t, router, http.MethodGet, "/listings", nil, ""))
		for _, item := range res.Items {
			if item.Listing.ID == listingID {
				return true
			}
		}
		return false
	}
	if !onMarket() {
		t.Fatal("the published listing should be on the marketplace")
	}

	// Suspending the farm takes its listings off the market and stops
	// reservations, without touching the listings themselves.
	if rec := review(reviewFarmRequest{Decision: "suspend", Reason: "Signalements répétés"}); rec.Code != http.StatusOK {
		t.Fatalf("suspend = %d; body = %s", rec.Code, rec.Body.String())
	}
	if onMarket() {
		t.Error("a suspended farm's listing is still on the marketplace")
	}
	buyer, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")
	if rec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{Quantity: 1}, buyer); rec.Code != http.StatusNotFound {
		t.Errorf("reserving from a suspended farm = %d, want 404", rec.Code)
	}
}

// MAX_FARMS_PER_ACCOUNT bounds how many farms an account opens; staff
// can raise it for one account; staff accounts themselves don't sell.
func TestFarm_HTTP_LimitPerAccountAndStaffOverride(t *testing.T) {
	router, tf := newTestRouterAndFakes()
	tf.createFarm.MaxFarms = 1
	root := adminToken(t, router, tf)
	token, userID := registerAndLogin(t, router, uniqueEmail("seller"), "farmer")
	create := func(name, tok string) *httptest.ResponseRecorder {
		return doJSON(t, router, http.MethodPost, "/farms", farmRequest{Name: name, Location: "Douala"}, tok)
	}

	if rec := create("Première", token); rec.Code != http.StatusCreated {
		t.Fatalf("first farm = %d; body = %s", rec.Code, rec.Body.String())
	}
	second := create("Deuxième", token)
	if second.Code != http.StatusConflict || decodeJSON[errorResponse](t, second).Code != "farm.limit_reached" {
		t.Fatalf("second farm at the default limit = %d %s, want 409 farm.limit_reached", second.Code, second.Body.String())
	}

	if rec := doJSON(t, router, http.MethodPost, "/admin/users/"+userID+"/farm-limit", farmLimitRequest{Limit: 2}, root); rec.Code != http.StatusOK {
		t.Fatalf("raise limit = %d; body = %s", rec.Code, rec.Body.String())
	}
	if rec := create("Deuxième", token); rec.Code != http.StatusCreated {
		t.Fatalf("second farm after staff raised the limit = %d; body = %s", rec.Code, rec.Body.String())
	}
	if rec := create("Troisième", token); rec.Code != http.StatusConflict {
		t.Fatalf("third farm = %d, want 409", rec.Code)
	}
	if farms := decodeJSON[[]farmResponse](t, doJSON(t, router, http.MethodGet, "/farms/mine", nil, token)); len(farms) != 2 {
		t.Errorf("farms/mine = %d farms, want 2", len(farms))
	}

	staffFarm := create("Ferme du staff", root)
	if staffFarm.Code != http.StatusForbidden || decodeJSON[errorResponse](t, staffFarm).Code != "farm.staff_cannot_sell" {
		t.Errorf("staff opening a farm = %d %s, want 403 farm.staff_cannot_sell", staffFarm.Code, staffFarm.Body.String())
	}
}

// FARM_AUTO_APPROVE lets a tester sell without staff in the loop: the
// farm is approved on submission, proof or not, and marked as such.
func TestFarm_HTTP_AutoApproveSwitch(t *testing.T) {
	router, tf := newTestRouterAndFakes()
	tf.submitFarm.AutoApprove = true
	token, farmID, listingID := newDraftFarmWithListing(t, router, tf)

	rec := doJSON(t, router, http.MethodPost, "/farms/"+farmID+"/verification/submit", submitFarmRequest{}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("submit = %d; body = %s", rec.Code, rec.Body.String())
	}
	farm := decodeJSON[farmResponse](t, rec)
	if farm.Status != "approved" || farm.Review == nil || !farm.Review.AutoApproved {
		t.Fatalf("after submit = %+v (review %+v), want approved and marked auto-approved", farm, farm.Review)
	}
	if rec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/publish", nil, token); rec.Code != http.StatusOK {
		t.Fatalf("publish = %d; body = %s", rec.Code, rec.Body.String())
	}
	// Not vouched for by a person: no verified badge until staff look.
	if profile := decodeJSON[farmProfileResponse](t, doJSON(t, router, http.MethodGet, "/farms/"+farmID, nil, "")); profile.Verified {
		t.Error("an auto-approved farm shows as verified")
	}
}

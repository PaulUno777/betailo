package httpserver

import (
	"net/http"
	"testing"
)

// adminToken registers an ordinary buyer, promotes them to admin
// directly in the fake store (POST /auth/register itself refuses the
// admin role — see TestRegister_HTTP_RejectsAdminRole), and logs in
// again so the issued token actually carries the new role.
func adminToken(t *testing.T, router http.Handler, tf *testFakes) string {
	t.Helper()
	_, userID := registerAndLogin(t, router, uniqueEmail("root"), "buyer")
	tf.promoteToAdmin(userID)

	loginRec := doJSON(t, router, http.MethodPost, "/auth/login", loginRequest{Email: emailForUserID(tf, userID), Password: "correct-horse-battery"}, "")
	if loginRec.Code != http.StatusOK {
		t.Fatalf("re-login after promotion status = %d, want %d", loginRec.Code, http.StatusOK)
	}
	return decodeJSON[loginResponse](t, loginRec).AccessToken
}

func emailForUserID(tf *testFakes, userID string) string {
	for email, u := range tf.users.byEmail {
		if u.ID.Hex() == userID {
			return email
		}
	}
	return ""
}

func TestAdmin_HTTP_RequiresAdminRole(t *testing.T) {
	router := newTestRouter()
	buyerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")

	rec := doJSON(t, router, http.MethodGet, "/admin/overview", nil, buyerToken)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin overview status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	anon := doJSON(t, router, http.MethodGet, "/admin/overview", nil, "")
	if anon.Code != http.StatusUnauthorized {
		t.Fatalf("no-token overview status = %d, want %d", anon.Code, http.StatusUnauthorized)
	}
}

func TestAdmin_HTTP_OverviewReflectsPlatformActivity(t *testing.T) {
	router, tf := newTestRouterAndFakes()
	listingID, _ := setupListing(t, router, 120)
	buyerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")
	doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{Quantity: 10, PayNow: true, PaymentMethod: "card"}, buyerToken)

	root := adminToken(t, router, tf)
	rec := doJSON(t, router, http.MethodGet, "/admin/overview", nil, root)
	if rec.Code != http.StatusOK {
		t.Fatalf("overview status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	out := decodeJSON[overviewResponse](t, rec)
	if out.ActiveCyclesPlanned != 1 {
		t.Errorf("ActiveCyclesPlanned = %d, want 1", out.ActiveCyclesPlanned)
	}
	wantVolume := 10 * 2500 * 1.02
	if out.VolumeReserved != wantVolume {
		t.Errorf("VolumeReserved = %v, want %v", out.VolumeReserved, wantVolume)
	}
	if out.EscrowHeld != wantVolume {
		t.Errorf("EscrowHeld = %v, want %v", out.EscrowHeld, wantVolume)
	}
}

func TestAdmin_HTTP_UsersFarmLimitAndSuspend(t *testing.T) {
	router, tf := newTestRouterAndFakes()
	root := adminToken(t, router, tf)
	targetToken, targetID := registerAndLogin(t, router, uniqueEmail("farmer"), "farmer")

	listRec := doJSON(t, router, http.MethodGet, "/admin/users", nil, root)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list users status = %d, want %d", listRec.Code, http.StatusOK)
	}
	if len(decodeJSON[[]adminUserResponse](t, listRec)) < 2 {
		t.Error("expected at least the admin and the farmer in the user list")
	}

	if rec := doJSON(t, router, http.MethodPost, "/admin/users/"+targetID+"/farm-limit", farmLimitRequest{Limit: 4}, root); rec.Code != http.StatusBadRequest || decodeJSON[errorResponse](t, rec).Code != "admin.farm_limit_reason_required" {
		t.Fatalf("farm-limit without a reason = %d %s, want 400 admin.farm_limit_reason_required", rec.Code, rec.Body.String())
	}
	limitRec := doJSON(t, router, http.MethodPost, "/admin/users/"+targetID+"/farm-limit", farmLimitRequest{Limit: 4, Reason: "Coopérative de 4 sites"}, root)
	if limitRec.Code != http.StatusOK {
		t.Fatalf("farm-limit status = %d, want %d; body = %s", limitRec.Code, http.StatusOK, limitRec.Body.String())
	}
	if got := decodeJSON[adminUserResponse](t, limitRec).FarmLimit; got != 4 {
		t.Errorf("FarmLimit = %d after raising it to 4", got)
	}
	if rec := doJSON(t, router, http.MethodPost, "/admin/users/"+targetID+"/farm-limit", farmLimitRequest{Limit: -1}, root); rec.Code != http.StatusBadRequest {
		t.Errorf("negative farm-limit status = %d, want 400", rec.Code)
	}
	if rec := doJSON(t, router, http.MethodPost, "/admin/users/"+targetID+"/farm-limit", farmLimitRequest{Limit: 2, Reason: "x"}, targetToken); rec.Code != http.StatusForbidden {
		t.Errorf("non-staff farm-limit status = %d, want 403", rec.Code)
	}

	suspendRec := doJSON(t, router, http.MethodPost, "/admin/users/"+targetID+"/suspend", nil, root)
	if suspendRec.Code != http.StatusOK {
		t.Fatalf("suspend status = %d, want %d; body = %s", suspendRec.Code, http.StatusOK, suspendRec.Body.String())
	}
	if !decodeJSON[adminUserResponse](t, suspendRec).Suspended {
		t.Error("Suspended = false after suspend")
	}

	// The session the user already had ends right away, not when its
	// access token would have expired.
	meRec := doJSON(t, router, http.MethodGet, "/me", nil, targetToken)
	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /me with the suspended user's open session = %d, want 401", meRec.Code)
	}
	if got := decodeJSON[errorResponse](t, meRec).Code; got != "account_suspended" {
		t.Errorf("code = %q, want account_suspended", got)
	}
}

func TestAdmin_HTTP_CyclesAndListingsOversight(t *testing.T) {
	router, tf := newTestRouterAndFakes()
	listingID, _ := setupListing(t, router, 120)
	root := adminToken(t, router, tf)

	cyclesRec := doJSON(t, router, http.MethodGet, "/admin/cycles", nil, root)
	if cyclesRec.Code != http.StatusOK || len(decodeJSON[[]cycleResponse](t, cyclesRec)) != 1 {
		t.Fatalf("admin cycles: status = %d, body = %s", cyclesRec.Code, cyclesRec.Body.String())
	}

	listingsRec := doJSON(t, router, http.MethodGet, "/admin/listings", nil, root)
	if listingsRec.Code != http.StatusOK {
		t.Fatalf("admin listings status = %d, want %d", listingsRec.Code, http.StatusOK)
	}
	found := false
	for _, l := range decodeJSON[[]listingResponse](t, listingsRec) {
		if l.ID == listingID {
			found = true
		}
	}
	if !found {
		t.Error("admin listings should include the created listing")
	}
}

func TestAdmin_HTTP_EscrowReleaseAndRefund(t *testing.T) {
	router, tf := newTestRouterAndFakes()
	listingID, farmerToken := setupListing(t, router, 120)
	buyerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")
	root := adminToken(t, router, tf)

	res1Rec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{Quantity: 5, PayNow: true, PaymentMethod: "card"}, buyerToken)
	res1 := decodeJSON[createReservationResponse](t, res1Rec).Reservation.ID
	doJSON(t, router, http.MethodPost, "/reservations/"+res1+"/mark-delivered", nil, farmerToken)

	res2Rec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{Quantity: 5, PayNow: true, PaymentMethod: "card"}, buyerToken)
	res2 := decodeJSON[createReservationResponse](t, res2Rec).Reservation.ID

	escrowRec := doJSON(t, router, http.MethodGet, "/admin/escrow", nil, root)
	if escrowRec.Code != http.StatusOK || len(decodeJSON[[]reservationResponse](t, escrowRec)) != 2 {
		t.Fatalf("admin escrow list: status = %d, body = %s", escrowRec.Code, escrowRec.Body.String())
	}

	releaseRec := doJSON(t, router, http.MethodPost, "/admin/escrow/"+res1+"/release", nil, root)
	if releaseRec.Code != http.StatusOK {
		t.Fatalf("release status = %d, want %d; body = %s", releaseRec.Code, http.StatusOK, releaseRec.Body.String())
	}
	if decodeJSON[reservationResponse](t, releaseRec).Payment.Status != "released" {
		t.Error("payment status should be released")
	}

	refundRec := doJSON(t, router, http.MethodPost, "/admin/escrow/"+res2+"/refund", nil, root)
	if refundRec.Code != http.StatusOK {
		t.Fatalf("refund status = %d, want %d; body = %s", refundRec.Code, http.StatusOK, refundRec.Body.String())
	}
	refunded := decodeJSON[reservationResponse](t, refundRec)
	if refunded.Payment.Status != "refunded" || refunded.Status != "cancelled" {
		t.Errorf("got = %+v, want refunded/cancelled", refunded)
	}
}

func TestAdmin_HTTP_ReportsDetailAndResolve(t *testing.T) {
	router, tf := newTestRouterAndFakes()
	_, farmerToken := setupListing(t, router, 120)
	_ = farmerToken
	reporterToken, _ := registerAndLogin(t, router, uniqueEmail("reporter"), "buyer")
	_, targetUserID := registerAndLogin(t, router, uniqueEmail("suspect"), "buyer")
	root := adminToken(t, router, tf)

	reportRec := doJSON(t, router, http.MethodPost, "/reports", createReportRequest{
		TargetID: targetUserID, TargetType: "profile", Reason: "usurpation",
	}, reporterToken)
	report := decodeJSON[reportResponse](t, reportRec)

	listRec := doJSON(t, router, http.MethodGet, "/admin/reports", nil, root)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list reports status = %d, want %d", listRec.Code, http.StatusOK)
	}

	detailRec := doJSON(t, router, http.MethodGet, "/admin/reports/"+report.ID, nil, root)
	if detailRec.Code != http.StatusOK {
		t.Fatalf("report detail status = %d, want %d; body = %s", detailRec.Code, http.StatusOK, detailRec.Body.String())
	}
	detail := decodeJSON[reportDetailResponse](t, detailRec)
	if detail.Report.ID != report.ID {
		t.Errorf("detail.Report.ID = %q, want %q", detail.Report.ID, report.ID)
	}

	resolveRec := doJSON(t, router, http.MethodPost, "/admin/reports/"+report.ID+"/resolve", resolveReportRequest{Action: "suspend"}, root)
	if resolveRec.Code != http.StatusOK {
		t.Fatalf("resolve status = %d, want %d; body = %s", resolveRec.Code, http.StatusOK, resolveRec.Body.String())
	}
	if decodeJSON[reportResponse](t, resolveRec).Status != "handled" {
		t.Error("report status should be handled after suspending the account")
	}
}

// Regression: GET /admin/listings once passed a nil storage port to
// toListingResponse, which panicked on the first listing that had photos.
func TestAdmin_HTTP_ListingsWithPhotosResolveURLs(t *testing.T) {
	router, tf := newTestRouterAndFakes()
	listingID, farmerToken := setupListing(t, router, 120)
	root := adminToken(t, router, tf)

	keys := make([]string, 0, 3)
	for range 3 {
		presignRec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/photos/presign", nil, farmerToken)
		if presignRec.Code != http.StatusOK {
			t.Fatalf("presign status = %d; body = %s", presignRec.Code, presignRec.Body.String())
		}
		keys = append(keys, decodeJSON[presignPhotoResponse](t, presignRec).Key)
	}
	setRec := doJSON(t, router, http.MethodPut, "/listings/"+listingID+"/photos", setListingPhotosRequest{Keys: keys}, farmerToken)
	if setRec.Code != http.StatusOK {
		t.Fatalf("set photos status = %d; body = %s", setRec.Code, setRec.Body.String())
	}

	rec := doJSON(t, router, http.MethodGet, "/admin/listings", nil, root)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin listings status = %d, want %d", rec.Code, http.StatusOK)
	}
	for _, l := range decodeJSON[[]listingResponse](t, rec) {
		if l.ID == listingID {
			if len(l.Photos) != len(keys) || l.Photos[0] != "https://fake-download.example/"+keys[0] {
				t.Fatalf("photos = %v, want signed download URLs for %v", l.Photos, keys)
			}
			return
		}
	}
	t.Fatal("admin listings should include the created listing")
}

// A listing report's detail scores the listing's owner: a brand-new,
// unverified farmer account trips both account signals.
func TestAdmin_HTTP_ListingReportScoresOwnerRisk(t *testing.T) {
	router, tf := newTestRouterAndFakes()
	tf.farms.awaitReview = true
	listingID, _ := setupListing(t, router, 120)
	reporterToken, _ := registerAndLogin(t, router, uniqueEmail("reporter"), "buyer")
	root := adminToken(t, router, tf)

	reportRec := doJSON(t, router, http.MethodPost, "/reports", createReportRequest{
		TargetID: listingID, TargetType: "listing", Reason: "photos volées",
	}, reporterToken)
	if reportRec.Code != http.StatusAccepted {
		t.Fatalf("create report status = %d; body = %s", reportRec.Code, reportRec.Body.String())
	}
	report := decodeJSON[reportResponse](t, reportRec)

	detailRec := doJSON(t, router, http.MethodGet, "/admin/reports/"+report.ID, nil, root)
	if detailRec.Code != http.StatusOK {
		t.Fatalf("report detail status = %d; body = %s", detailRec.Code, detailRec.Body.String())
	}
	detail := decodeJSON[reportDetailResponse](t, detailRec)
	names := map[string]bool{}
	for _, s := range detail.Signals {
		names[s.Name] = true
	}
	if !names["Compte très récent"] || !names["Ferme non vérifiée"] {
		t.Fatalf("signals = %+v, want the new-account and unreviewed-farm signals", detail.Signals)
	}
	if detail.RiskScore <= 0 || detail.RiskScore > 100 {
		t.Fatalf("riskScore = %d, want in (0, 100]", detail.RiskScore)
	}
}

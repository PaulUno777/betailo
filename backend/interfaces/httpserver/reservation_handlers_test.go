package httpserver

import (
	"net/http"
	"slices"
	"testing"
	"time"
)

// setupListing registers a farmer, creates a farm, a cycle and a
// published listing on it — the shared starting point for every
// reservation/escrow test below.
func setupListing(t *testing.T, router http.Handler, capacity int) (listingID, farmerToken string) {
	t.Helper()
	farmerToken, _ = registerAndLogin(t, router, uniqueEmail("farmer"), "farmer")

	farmRec := doJSON(t, router, http.MethodPost, "/farms", farmRequest{Name: "Ferme des Collines", Location: "Dschang"}, farmerToken)
	farmOut := decodeJSON[farmResponse](t, farmRec)

	cycleRec := doJSON(t, router, http.MethodPost, "/cycles", createCycleRequest{
		FarmID: farmOut.ID, Species: "poule", Type: "couvee", Capacity: capacity, StartsAt: time.Now().Add(24 * time.Hour),
	}, farmerToken)
	cycleOut := decodeJSON[cycleResponse](t, cycleRec)

	listingRec := doJSON(t, router, http.MethodPost, "/listings", createListingRequest{
		CycleID: cycleOut.ID, Title: "Poussins Brahma", UnitPrice: 2500,
	}, farmerToken)
	if listingRec.Code != http.StatusCreated {
		t.Fatalf("create listing status = %d, want %d; body = %s", listingRec.Code, http.StatusCreated, listingRec.Body.String())
	}
	return decodeJSON[listingResponse](t, listingRec).ID, farmerToken
}

func TestReservation_HTTP_PayNow_IsPriorityAndHeld(t *testing.T) {
	router := newTestRouter()
	listingID, _ := setupListing(t, router, 120)
	buyerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")

	rec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{
		Quantity: 30, PayNow: true, PaymentMethod: "mtn_momo", PayerPhone: "690123456",
	}, buyerToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("reserve status = %d, want %d; body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	out := decodeJSON[createReservationResponse](t, rec)
	if !out.PaymentCaptured {
		t.Error("PaymentCaptured = false, want true")
	}
	if !out.Reservation.Priority {
		t.Error("Priority = false, want true for a paid reservation")
	}
	if out.Reservation.Status != "confirmed" {
		t.Errorf("Status = %q, want %q", out.Reservation.Status, "confirmed")
	}
	if out.Reservation.Payment == nil || out.Reservation.Payment.Status != "held" {
		t.Errorf("Payment = %+v, want held", out.Reservation.Payment)
	}
	wantFee := 30 * 2500 * 0.02
	if out.Reservation.ServiceFee != wantFee {
		t.Errorf("ServiceFee = %v, want %v", out.Reservation.ServiceFee, wantFee)
	}
}

func TestReservation_HTTP_PayLater_JoinsTheWaitingList(t *testing.T) {
	router := newTestRouter()
	listingID, _ := setupListing(t, router, 120)
	buyerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")

	rec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{Quantity: 10}, buyerToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("reserve status = %d, want %d; body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	out := decodeJSON[createReservationResponse](t, rec)
	if out.PaymentCaptured {
		t.Error("PaymentCaptured = true, want false: PayNow wasn't requested")
	}
	if out.Reservation.Priority {
		t.Error("Priority = true, want false for an unpaid reservation")
	}
	if out.Reservation.Status != "pending" {
		t.Errorf("Status = %q, want %q", out.Reservation.Status, "pending")
	}

	// Unpaid reservations hold no places: a paying buyer can take all 120.
	payerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")
	paid := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{
		Quantity: 120, PayNow: true, PaymentMethod: "card",
	}, payerToken)
	if paid.Code != http.StatusCreated {
		t.Fatalf("paid reservation of every place: status = %d, want 201; body = %s", paid.Code, paid.Body.String())
	}

	// The waiting buyer can no longer pay: the places went to a payer.
	payRec := doJSON(t, router, http.MethodPost, "/reservations/"+out.Reservation.ID+"/pay", payReservationRequest{PaymentMethod: "card"}, buyerToken)
	if payRec.Code != http.StatusConflict {
		t.Fatalf("pay when sold out: status = %d, want 409; body = %s", payRec.Code, payRec.Body.String())
	}
	if code := decodeJSON[errorResponse](t, payRec).Code; code != "reservation.sold_out" {
		t.Errorf("code = %q, want reservation.sold_out", code)
	}
}

func TestReservation_HTTP_ExceedingCapacity_Returns409(t *testing.T) {
	router := newTestRouter()
	listingID, _ := setupListing(t, router, 5)
	buyerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")

	rec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{
		Quantity: 6, PayNow: true, PaymentMethod: "card",
	}, buyerToken)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestReservation_HTTP_FullEscrowLifecycle(t *testing.T) {
	router := newTestRouter()
	listingID, farmerToken := setupListing(t, router, 120)
	buyerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")

	reserveRec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{
		Quantity: 20, PayNow: true, PaymentMethod: "card",
	}, buyerToken)
	reservationID := decodeJSON[createReservationResponse](t, reserveRec).Reservation.ID

	// Only the farm's owner can mark it delivered.
	forbidden := doJSON(t, router, http.MethodPost, "/reservations/"+reservationID+"/mark-delivered", nil, buyerToken)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("buyer mark-delivered status = %d, want %d", forbidden.Code, http.StatusForbidden)
	}

	deliverRec := doJSON(t, router, http.MethodPost, "/reservations/"+reservationID+"/mark-delivered", nil, farmerToken)
	if deliverRec.Code != http.StatusOK {
		t.Fatalf("mark-delivered status = %d, want %d; body = %s", deliverRec.Code, http.StatusOK, deliverRec.Body.String())
	}
	if decodeJSON[reservationResponse](t, deliverRec).Status != "delivered" {
		t.Error("status after mark-delivered should be delivered")
	}

	// Only the buyer can confirm.
	if rec := doJSON(t, router, http.MethodPost, "/reservations/"+reservationID+"/confirm-delivery", nil, farmerToken); rec.Code != http.StatusForbidden {
		t.Fatalf("farmer confirm-delivery status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	confirmRec := doJSON(t, router, http.MethodPost, "/reservations/"+reservationID+"/confirm-delivery", nil, buyerToken)
	if confirmRec.Code != http.StatusOK {
		t.Fatalf("confirm-delivery status = %d, want %d; body = %s", confirmRec.Code, http.StatusOK, confirmRec.Body.String())
	}
	confirmed := decodeJSON[reservationResponse](t, confirmRec)
	if confirmed.Payment == nil || confirmed.Payment.Status != "released" {
		t.Errorf("Payment = %+v, want released", confirmed.Payment)
	}

	// The buyer's own reservation list should show it.
	mineRec := doJSON(t, router, http.MethodGet, "/reservations/mine", nil, buyerToken)
	mine := decodeJSON[reservationListResponse](t, mineRec)
	found := false
	for _, r := range mine.Items {
		if r.ID == reservationID {
			found = true
			if r.Group != "done" || !slices.Contains(r.Actions, "review") || r.Listing.Title == "" || r.Reference == "" {
				t.Errorf("released reservation view = %+v, want done group, a review action, listing title and reference", r)
			}
		}
	}
	if mine.Counts["done"] != 1 || mine.EscrowTotal != 0 {
		t.Errorf("counts=%v escrowTotal=%v, want one done and nothing held", mine.Counts, mine.EscrowTotal)
	}
	if !found {
		t.Error("GET /reservations/mine doesn't include the buyer's own reservation")
	}
}

func TestReservation_HTTP_DisputeBlocksConfirmDelivery(t *testing.T) {
	router := newTestRouter()
	listingID, farmerToken := setupListing(t, router, 120)
	buyerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")

	reserveRec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{Quantity: 10, PayNow: true, PaymentMethod: "card"}, buyerToken)
	reservationID := decodeJSON[createReservationResponse](t, reserveRec).Reservation.ID
	doJSON(t, router, http.MethodPost, "/reservations/"+reservationID+"/mark-delivered", nil, farmerToken)

	disputeRec := doJSON(t, router, http.MethodPost, "/reservations/"+reservationID+"/dispute", openDisputeRequest{Reason: "poussins manquants"}, buyerToken)
	if disputeRec.Code != http.StatusAccepted {
		t.Fatalf("dispute status = %d, want %d; body = %s", disputeRec.Code, http.StatusAccepted, disputeRec.Body.String())
	}

	confirmRec := doJSON(t, router, http.MethodPost, "/reservations/"+reservationID+"/confirm-delivery", nil, buyerToken)
	if confirmRec.Code != http.StatusConflict {
		t.Fatalf("confirm-delivery with open dispute: status = %d, want %d", confirmRec.Code, http.StatusConflict)
	}
}

func TestReservation_HTTP_PayLaterThenPay(t *testing.T) {
	router := newTestRouter()
	listingID, _ := setupListing(t, router, 120)
	buyerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")

	reserveRec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{Quantity: 10}, buyerToken)
	reservationID := decodeJSON[createReservationResponse](t, reserveRec).Reservation.ID

	payRec := doJSON(t, router, http.MethodPost, "/reservations/"+reservationID+"/pay", payReservationRequest{PaymentMethod: "orange_money", PayerPhone: "690123456"}, buyerToken)
	if payRec.Code != http.StatusOK {
		t.Fatalf("pay status = %d, want %d; body = %s", payRec.Code, http.StatusOK, payRec.Body.String())
	}
	paid := decodeJSON[reservationResponse](t, payRec)
	if paid.Status != "confirmed" || !paid.Priority || paid.Payment.Status != "held" {
		t.Errorf("got = %+v, want confirmed/priority/held", paid)
	}
}

func TestReservation_HTTP_ListByFarm_OwnerOnly(t *testing.T) {
	router, tf := newTestRouterAndFakes()
	listingID, farmerToken := setupListing(t, router, 120)
	buyerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")
	doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{Quantity: 5, PayNow: true, PaymentMethod: "card"}, buyerToken)

	var farmID string
	for _, f := range tf.farms.byID {
		farmID = f.ID.Hex()
	}

	ownerRec := doJSON(t, router, http.MethodGet, "/farms/"+farmID+"/reservations", nil, farmerToken)
	if ownerRec.Code != http.StatusOK {
		t.Fatalf("owner list status = %d, want %d; body = %s", ownerRec.Code, http.StatusOK, ownerRec.Body.String())
	}
	owned := decodeJSON[reservationListResponse](t, ownerRec)
	if len(owned.Items) != 1 {
		t.Fatal("farm owner should see exactly one reservation")
	}
	if !slices.Contains(owned.Items[0].Actions, "mark_delivered") || owned.EscrowTotal != owned.Items[0].TotalAmount {
		t.Errorf("farmer view = %+v (escrow %v), want mark_delivered offered and the held amount totalled", owned.Items[0].Actions, owned.EscrowTotal)
	}

	otherFarmerToken, _ := registerAndLogin(t, router, uniqueEmail("other-farmer"), "farmer")
	forbiddenRec := doJSON(t, router, http.MethodGet, "/farms/"+farmID+"/reservations", nil, otherFarmerToken)
	if forbiddenRec.Code != http.StatusForbidden {
		t.Fatalf("non-owner list status = %d, want %d", forbiddenRec.Code, http.StatusForbidden)
	}
}

func TestReservation_HTTP_ActionsFollowTheLifecycleAndCancel(t *testing.T) {
	router := newTestRouter()
	listingID, farmerToken := setupListing(t, router, 120)
	buyerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")

	// Farmers can't reserve their own listing.
	own := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{Quantity: 1}, farmerToken)
	if own.Code != http.StatusForbidden {
		t.Fatalf("own listing status = %d, want 403", own.Code)
	}

	rec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{Quantity: 4}, buyerToken)
	id := decodeJSON[createReservationResponse](t, rec).Reservation.ID

	list := decodeJSON[reservationListResponse](t, doJSON(t, router, http.MethodGet, "/reservations/mine?group=active", nil, buyerToken))
	if len(list.Items) != 1 || !slices.Contains(list.Items[0].Actions, "pay") || !slices.Contains(list.Items[0].Actions, "cancel") {
		t.Fatalf("unpaid view = %+v, want pay and cancel offered", list.Items)
	}

	cancel := doJSON(t, router, http.MethodPost, "/reservations/"+id+"/cancel", nil, buyerToken)
	if cancel.Code != http.StatusOK || decodeJSON[reservationResponse](t, cancel).Status != "cancelled" {
		t.Fatalf("cancel status = %d; body = %s", cancel.Code, cancel.Body.String())
	}
	list = decodeJSON[reservationListResponse](t, doJSON(t, router, http.MethodGet, "/reservations/mine?group=cancelled", nil, buyerToken))
	if len(list.Items) != 1 || len(list.Items[0].Actions) != 0 || list.Counts["active"] != 0 {
		t.Fatalf("after cancel = %+v counts=%v, want it under cancelled with no actions", list.Items, list.Counts)
	}

	if bad := doJSON(t, router, http.MethodGet, "/reservations/mine?group=bogus", nil, buyerToken); bad.Code != http.StatusBadRequest {
		t.Errorf("unknown group status = %d, want 400", bad.Code)
	}
}

func TestReservation_HTTP_QuoteMatchesTheReservation(t *testing.T) {
	router := newTestRouter()
	listingID, farmerToken := setupListing(t, router, 120)
	buyerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")

	rec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/quote", quoteRequest{Quantity: 30}, buyerToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("quote status = %d; body = %s", rec.Code, rec.Body.String())
	}
	q := decodeJSON[quoteResponse](t, rec)
	if q.Subtotal != 75000 || q.ServiceFee != 1500 || q.Total != 76500 || q.Max != 120 || q.Min != 1 {
		t.Fatalf("quote = %+v, want 30 x 2500 + 2%% fee, bounds 1..120", q)
	}

	res := decodeJSON[createReservationResponse](t, doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations",
		createReservationRequest{Quantity: 30, PayNow: true, PaymentMethod: "card"}, buyerToken)).Reservation
	if res.TotalAmount != q.Total || res.ServiceFee != q.ServiceFee {
		t.Errorf("reservation total %v/fee %v, want the quoted %v/%v", res.TotalAmount, res.ServiceFee, q.Total, q.ServiceFee)
	}

	if rec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/quote", quoteRequest{Quantity: 91, PayNow: true}, buyerToken); rec.Code != http.StatusConflict {
		t.Errorf("quote over what's left: status = %d, want 409", rec.Code)
	}
	// The waiting list only has to fit the cycle, not what's left.
	if rec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/quote", quoteRequest{Quantity: 91}, buyerToken); rec.Code != http.StatusOK {
		t.Errorf("waiting-list quote over what's left: status = %d, want 200", rec.Code)
	}
	if rec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/quote", quoteRequest{Quantity: 1}, farmerToken); rec.Code != http.StatusForbidden {
		t.Errorf("quote on your own listing: status = %d, want 403", rec.Code)
	}
}

func TestNavCounts_HTTP_CountWhatWaitsOnEachSide(t *testing.T) {
	router := newTestRouter()
	listingID, farmerToken := setupListing(t, router, 120)
	buyerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")

	counts := func(token string) navCountsResponse {
		t.Helper()
		rec := doJSON(t, router, http.MethodGet, "/me/nav-counts", nil, token)
		if rec.Code != http.StatusOK {
			t.Fatalf("nav-counts status = %d; body = %s", rec.Code, rec.Body.String())
		}
		return decodeJSON[navCountsResponse](t, rec)
	}

	if got := counts(buyerToken); got != (navCountsResponse{}) {
		t.Fatalf("before any reservation: %+v, want zeros", got)
	}

	// Unpaid: the buyer has to pay; nothing for the farmer to do yet.
	doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{Quantity: 5}, buyerToken)
	// Paid: the farmer has to deliver.
	doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{
		Quantity: 5, PayNow: true, PaymentMethod: "mtn_momo", PayerPhone: "690123456",
	}, buyerToken)

	if got := counts(buyerToken); got.Buyer != 1 || got.Seller != 0 {
		t.Errorf("buyer counts = %+v, want {Buyer:1 Seller:0}", got)
	}
	if got := counts(farmerToken); got.Buyer != 0 || got.Seller != 1 {
		t.Errorf("farmer counts = %+v, want {Buyer:0 Seller:1}", got)
	}

	if rec := doJSON(t, router, http.MethodGet, "/me/nav-counts", nil, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous status = %d, want 401", rec.Code)
	}
}

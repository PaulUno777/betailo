package httpserver

import (
	"net/http"
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

func TestReservation_HTTP_PayLater_IsUnpaidWithDeadline(t *testing.T) {
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
	if out.Reservation.PaymentDeadline == nil {
		t.Error("PaymentDeadline is nil, want a deadline ~24h out")
	}
}

func TestReservation_HTTP_ExceedingCapacity_Returns409(t *testing.T) {
	router := newTestRouter()
	listingID, _ := setupListing(t, router, 5)
	buyerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")

	rec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{Quantity: 6}, buyerToken)
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
	mine := decodeJSON[[]reservationResponse](t, mineRec)
	found := false
	for _, r := range mine {
		if r.ID == reservationID {
			found = true
		}
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
	if len(decodeJSON[[]reservationResponse](t, ownerRec)) != 1 {
		t.Error("farm owner should see exactly one reservation")
	}

	otherFarmerToken, _ := registerAndLogin(t, router, uniqueEmail("other-farmer"), "farmer")
	forbiddenRec := doJSON(t, router, http.MethodGet, "/farms/"+farmID+"/reservations", nil, otherFarmerToken)
	if forbiddenRec.Code != http.StatusForbidden {
		t.Fatalf("non-owner list status = %d, want %d", forbiddenRec.Code, http.StatusForbidden)
	}
}

package httpserver

import (
	"net/http"
	"testing"
)

func TestSellerDashboard_HTTP_ReflectsReservationsAndEscrow(t *testing.T) {
	router, tf := newTestRouterAndFakes()
	listingID, farmerToken := setupListing(t, router, 120)
	buyerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")

	doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{Quantity: 20, PayNow: true, PaymentMethod: "card"}, buyerToken)

	var farmID string
	for _, f := range tf.farms.byID {
		farmID = f.ID.Hex()
	}

	rec := doJSON(t, router, http.MethodGet, "/farms/"+farmID+"/dashboard", nil, farmerToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("dashboard status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	out := decodeJSON[sellerDashboardResponse](t, rec)
	if out.ReservationsThisPeriod != 1 {
		t.Errorf("ReservationsThisPeriod = %d, want 1", out.ReservationsThisPeriod)
	}
	wantHeld := 20 * 2500 * 1.02
	if out.EscrowHeldTotal != wantHeld {
		t.Errorf("EscrowHeldTotal = %v, want %v", out.EscrowHeldTotal, wantHeld)
	}

	buyerRec := doJSON(t, router, http.MethodGet, "/farms/"+farmID+"/dashboard", nil, buyerToken)
	if buyerRec.Code != http.StatusForbidden {
		t.Fatalf("non-owner dashboard status = %d, want %d", buyerRec.Code, http.StatusForbidden)
	}
}

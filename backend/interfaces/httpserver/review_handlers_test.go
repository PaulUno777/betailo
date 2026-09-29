package httpserver

import (
	"net/http"
	"testing"
)

// reserveAndRelease drives a reservation all the way to released escrow
// — the only state application/usecase/review.CreateReview accepts —
// and returns its ID.
func reserveAndRelease(t *testing.T, router http.Handler, listingID, farmerToken, buyerToken string) string {
	t.Helper()
	reserveRec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{Quantity: 10, PayNow: true}, buyerToken)
	reservationID := decodeJSON[createReservationResponse](t, reserveRec).Reservation.ID
	doJSON(t, router, http.MethodPost, "/reservations/"+reservationID+"/mark-delivered", nil, farmerToken)
	doJSON(t, router, http.MethodPost, "/reservations/"+reservationID+"/confirm-delivery", nil, buyerToken)
	return reservationID
}

func TestReview_HTTP_CreateAfterReleaseThenSummary(t *testing.T) {
	router, tf := newTestRouterAndFakes()
	listingID, farmerToken := setupListing(t, router, 120)
	buyerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")
	reservationID := reserveAndRelease(t, router, listingID, farmerToken, buyerToken)

	rec := doJSON(t, router, http.MethodPost, "/reservations/"+reservationID+"/reviews", createReviewRequest{
		Rating: 5, Comment: "60 poussins reçus, 58 bien vivants.",
	}, buyerToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create review status = %d, want %d; body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	// A second review on the same reservation must be rejected (one
	// review per reservation).
	dup := doJSON(t, router, http.MethodPost, "/reservations/"+reservationID+"/reviews", createReviewRequest{Rating: 1}, buyerToken)
	if dup.Code != http.StatusConflict {
		t.Fatalf("duplicate review status = %d, want %d", dup.Code, http.StatusConflict)
	}

	var farmID string
	for _, f := range tf.farms.byID {
		farmID = f.ID.Hex()
	}
	summaryRec := doJSON(t, router, http.MethodGet, "/farms/"+farmID+"/reviews", nil, "")
	if summaryRec.Code != http.StatusOK {
		t.Fatalf("list reviews status = %d, want %d", summaryRec.Code, http.StatusOK)
	}
	out := decodeJSON[farmReviewsResponse](t, summaryRec)
	if out.Summary.Count != 1 || out.Summary.Average != 5 {
		t.Errorf("Summary = %+v, want count=1 average=5", out.Summary)
	}
	if len(out.Reviews) != 1 || out.Reviews[0].Rating != 5 {
		t.Errorf("Reviews = %+v, want one 5-star review", out.Reviews)
	}
}

func TestReview_HTTP_RejectsBeforeEscrowReleased(t *testing.T) {
	router := newTestRouter()
	listingID, farmerToken := setupListing(t, router, 120)
	buyerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")

	reserveRec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{Quantity: 5, PayNow: true}, buyerToken)
	reservationID := decodeJSON[createReservationResponse](t, reserveRec).Reservation.ID
	_ = farmerToken

	rec := doJSON(t, router, http.MethodPost, "/reservations/"+reservationID+"/reviews", createReviewRequest{Rating: 5}, buyerToken)
	if rec.Code != http.StatusConflict {
		t.Fatalf("review before release status = %d, want %d; body = %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

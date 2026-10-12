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
	reserveRec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{Quantity: 10, PayNow: true, PaymentMethod: "card"}, buyerToken)
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
	if len(out.Items) != 1 || out.Items[0].Rating != 5 || out.Total != 1 {
		t.Errorf("Items = %+v, want one 5-star review", out.Items)
	}
}

func TestReview_HTTP_RejectsBeforeEscrowReleased(t *testing.T) {
	router := newTestRouter()
	listingID, farmerToken := setupListing(t, router, 120)
	buyerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")

	reserveRec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{Quantity: 5, PayNow: true, PaymentMethod: "card"}, buyerToken)
	reservationID := decodeJSON[createReservationResponse](t, reserveRec).Reservation.ID
	_ = farmerToken

	rec := doJSON(t, router, http.MethodPost, "/reservations/"+reservationID+"/reviews", createReviewRequest{Rating: 5}, buyerToken)
	if rec.Code != http.StatusConflict {
		t.Fatalf("review before release status = %d, want %d; body = %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestReview_HTTP_RichReviewHelpfulAndReply(t *testing.T) {
	router, tf := newTestRouterAndFakes()
	listingID, farmerToken := setupListing(t, router, 120)
	buyerToken, buyerID := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")
	tf.setDisplayName(buyerID, "Joël Tchoupo")
	reservationID := reserveAndRelease(t, router, listingID, farmerToken, buyerToken)

	presign := doJSON(t, router, http.MethodPost, "/reservations/"+reservationID+"/reviews/photos/presign", nil, buyerToken)
	if presign.Code != http.StatusOK {
		t.Fatalf("presign status = %d; body = %s", presign.Code, presign.Body.String())
	}
	key := decodeJSON[presignPhotoResponse](t, presign).Key

	// A key that wasn't issued for this reservation is refused.
	foreign := doJSON(t, router, http.MethodPost, "/reservations/"+reservationID+"/reviews", createReviewRequest{
		Rating: 5, PhotoKeys: []string{"listings/6abb0f3dcd905c20e75b87c9/6ac3c1e70c7615f679083e6d.jpg"},
	}, buyerToken)
	if foreign.Code != http.StatusBadRequest {
		t.Fatalf("foreign photo key status = %d, want 400", foreign.Code)
	}
	if bad := doJSON(t, router, http.MethodPost, "/reservations/"+reservationID+"/reviews", createReviewRequest{Rating: 5, Tags: []string{"free_beer"}}, buyerToken); bad.Code != http.StatusBadRequest {
		t.Fatalf("unknown tag status = %d, want 400", bad.Code)
	}

	rec := doJSON(t, router, http.MethodPost, "/reservations/"+reservationID+"/reviews", createReviewRequest{
		Rating: 4, Comment: "Très bonne souche.", Criteria: criteriaJSON{Health: 5, Conformity: 4, Communication: 5},
		Tags: []string{"healthy_animals", "responsive_farmer"}, PhotoKeys: []string{key},
	}, buyerToken)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create rich review status = %d; body = %s", rec.Code, rec.Body.String())
	}
	reviewID := decodeJSON[reviewResponse](t, rec).ID

	var farmID string
	for _, f := range tf.farms.byID {
		farmID = f.ID.Hex()
	}
	page := decodeJSON[farmReviewsResponse](t, doJSON(t, router, http.MethodGet, "/farms/"+farmID+"/reviews?filter=photos", nil, ""))
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("photos filter = %+v, want the one review with a photo", page)
	}
	it := page.Items[0]
	if it.AuthorName != "Joël T." || it.ReservationRef == "" || len(it.Photos) != 1 || it.Criteria.Health != 5 || len(it.Tags) != 2 {
		t.Errorf("review item = %+v, want public name, reservation ref, photo, criteria and tags", it)
	}
	if page.Summary.Criteria.Health != 5 || page.Summary.Criteria.Timeliness != 0 {
		t.Errorf("criteria averages = %+v, want health 5 and timeliness unrated", page.Summary.Criteria)
	}
	if five := decodeJSON[farmReviewsResponse](t, doJSON(t, router, http.MethodGet, "/farms/"+farmID+"/reviews?filter=5", nil, "")); five.Total != 0 {
		t.Errorf("5-star filter total = %d, want 0 for a 4-star review", five.Total)
	}
	if bad := doJSON(t, router, http.MethodGet, "/farms/"+farmID+"/reviews?filter=bogus", nil, ""); bad.Code != http.StatusBadRequest {
		t.Errorf("unknown filter status = %d, want 400", bad.Code)
	}

	// Helpful: idempotent per user, never on your own review.
	otherToken, _ := registerAndLogin(t, router, uniqueEmail("other"), "buyer")
	for range 2 {
		h := doJSON(t, router, http.MethodPost, "/reviews/"+reviewID+"/helpful", nil, otherToken)
		if h.Code != http.StatusOK || decodeJSON[helpfulResponse](t, h).HelpfulCount != 1 {
			t.Fatalf("helpful vote: status = %d body = %s, want count 1 both times", h.Code, h.Body.String())
		}
	}
	if own := doJSON(t, router, http.MethodPost, "/reviews/"+reviewID+"/helpful", nil, buyerToken); own.Code != http.StatusForbidden {
		t.Errorf("voting your own review status = %d, want 403", own.Code)
	}
	mine := decodeJSON[farmReviewsResponse](t, doJSON(t, router, http.MethodGet, "/farms/"+farmID+"/reviews", nil, otherToken))
	if !mine.Items[0].VotedByMe || mine.Items[0].Mine {
		t.Error("for the voter: votedByMe should be true and mine false")
	}
	if own := decodeJSON[farmReviewsResponse](t, doJSON(t, router, http.MethodGet, "/farms/"+farmID+"/reviews", nil, buyerToken)); !own.Items[0].Mine {
		t.Error("mine should be true for the review's author")
	}

	// Reply: owner only, once, and the reviewer hears about it.
	if r := doJSON(t, router, http.MethodPost, "/reviews/"+reviewID+"/reply", reviewReplyRequest{Text: "Merci !"}, otherToken); r.Code != http.StatusForbidden {
		t.Errorf("non-owner reply status = %d, want 403", r.Code)
	}
	reply := doJSON(t, router, http.MethodPost, "/reviews/"+reviewID+"/reply", reviewReplyRequest{Text: "Merci Joël, la pluie avait coupé la route."}, farmerToken)
	if reply.Code != http.StatusOK || decodeJSON[reviewResponse](t, reply).Reply == nil {
		t.Fatalf("owner reply status = %d; body = %s", reply.Code, reply.Body.String())
	}
	if again := doJSON(t, router, http.MethodPost, "/reviews/"+reviewID+"/reply", reviewReplyRequest{Text: "Encore"}, farmerToken); again.Code != http.StatusConflict {
		t.Errorf("second reply status = %d, want 409", again.Code)
	}
	n := notificationsOf(t, router, buyerToken)
	found := false
	for _, x := range n.Items {
		found = found || x.Type == "review_reply"
	}
	if !found {
		t.Errorf("buyer notifications = %+v, want a review_reply", n.Items)
	}
}

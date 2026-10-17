package httpserver

import (
	"net/http"
	"testing"
)

func notificationsOf(t *testing.T, router http.Handler, token string) notificationListResponse {
	t.Helper()
	rec := doJSON(t, router, http.MethodGet, "/notifications", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /notifications status = %d", rec.Code)
	}
	return decodeJSON[notificationListResponse](t, rec)
}

func TestNotifications_HTTP_EachSideIsToldAndCanMarkRead(t *testing.T) {
	router := newTestRouter()
	listingID, farmerToken := setupListing(t, router, 120)
	buyerToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")

	rec := doJSON(t, router, http.MethodPost, "/listings/"+listingID+"/reservations", createReservationRequest{
		Quantity: 3, PayNow: true, PaymentMethod: "mtn_momo", PayerPhone: "690123456",
	}, buyerToken)
	res := decodeJSON[createReservationResponse](t, rec).Reservation

	farmer := notificationsOf(t, router, farmerToken)
	if farmer.Unread != 1 || len(farmer.Items) != 1 || farmer.Items[0].Type != "reservation_paid" ||
		farmer.Items[0].Params["ref"] != res.Reference || farmer.Items[0].Link != "/portal/reservations" {
		t.Fatalf("farmer notifications = %+v, want one unread reservation_paid with the reference", farmer)
	}

	if rec := doJSON(t, router, http.MethodPost, "/reservations/"+res.ID+"/mark-delivered", nil, farmerToken); rec.Code != http.StatusOK {
		t.Fatalf("mark delivered status = %d", rec.Code)
	}
	buyer := notificationsOf(t, router, buyerToken)
	if len(buyer.Items) != 1 || buyer.Items[0].Type != "reservation_delivered" || buyer.Items[0].Params["hours"] != "48" {
		t.Fatalf("buyer notifications = %+v, want reservation_delivered with the 48h window", buyer.Items)
	}

	// Nobody can mark someone else's notification read.
	if rec := doJSON(t, router, http.MethodPost, "/notifications/"+buyer.Items[0].ID+"/read", nil, farmerToken); rec.Code != http.StatusNotFound {
		t.Errorf("marking another user's notification: status = %d, want 404", rec.Code)
	}
	if rec := doJSON(t, router, http.MethodPost, "/notifications/"+buyer.Items[0].ID+"/read", nil, buyerToken); rec.Code != http.StatusNoContent {
		t.Fatalf("mark read status = %d", rec.Code)
	}
	count := decodeJSON[unreadCountResponse](t, doJSON(t, router, http.MethodGet, "/notifications/unread-count", nil, buyerToken))
	if count.Unread != 0 {
		t.Errorf("buyer unread after mark read = %d, want 0", count.Unread)
	}

	if rec := doJSON(t, router, http.MethodPost, "/notifications/read-all", nil, farmerToken); rec.Code != http.StatusNoContent {
		t.Fatalf("read-all status = %d", rec.Code)
	}
	if n := notificationsOf(t, router, farmerToken); n.Unread != 0 || !n.Items[0].Read {
		t.Errorf("farmer after read-all = %+v, want everything read", n)
	}
}

func TestNotifications_HTTP_ForumReplyTellsTheAuthorButNotYourself(t *testing.T) {
	router := newTestRouter()
	authorToken, _ := registerAndLogin(t, router, uniqueEmail("author"), "farmer")
	replierToken, _ := registerAndLogin(t, router, uniqueEmail("replier"), "buyer")

	post := decodeJSON[forumPostResponse](t, doJSON(t, router, http.MethodPost, "/forum/posts", createForumPostRequest{
		Category: "health", Title: "Vaccin Newcastle", Body: "Quel calendrier ?",
	}, authorToken))

	doJSON(t, router, http.MethodPost, "/forum/posts/"+post.ID+"/replies", map[string]string{"body": "Je réponds à moi-même"}, authorToken)
	if n := notificationsOf(t, router, authorToken); len(n.Items) != 0 {
		t.Fatalf("replying to your own post notified you: %+v", n.Items)
	}

	doJSON(t, router, http.MethodPost, "/forum/posts/"+post.ID+"/replies", map[string]string{"body": "J+7 puis J+21"}, replierToken)
	n := notificationsOf(t, router, authorToken)
	if len(n.Items) != 1 || n.Items[0].Type != "forum_reply" || n.Items[0].Link != "/forum/"+post.ID {
		t.Fatalf("author notifications = %+v, want one forum_reply linking to the post", n.Items)
	}
}

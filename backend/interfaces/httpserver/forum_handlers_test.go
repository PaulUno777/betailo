package httpserver

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pauluno777/betailo/backend/domain"
)

func TestForum_HTTP_PostReplyAndMarkBestAnswer(t *testing.T) {
	router := newTestRouter()
	authorToken, _ := registerAndLogin(t, router, uniqueEmail("farmer"), "farmer")
	replierToken, _ := registerAndLogin(t, router, uniqueEmail("buyer"), "buyer")

	postRec := doJSON(t, router, http.MethodPost, "/forum/posts", createForumPostRequest{
		Category: "health", Title: "Calendrier de vaccination ?", Body: "500 poulets, quel calendrier ?",
	}, authorToken)
	if postRec.Code != http.StatusCreated {
		t.Fatalf("create post status = %d, want %d; body = %s", postRec.Code, http.StatusCreated, postRec.Body.String())
	}
	post := decodeJSON[forumPostResponse](t, postRec)

	odd := doJSON(t, router, http.MethodPost, "/forum/posts", createForumPostRequest{Category: "n'importe quoi", Title: "Titre", Body: "Texte"}, authorToken)
	if odd.Code != http.StatusBadRequest || decodeJSON[errorResponse](t, odd).Code != "forum.category_unknown" {
		t.Errorf("unlisted category: status = %d, body = %s; want 400 forum.category_unknown", odd.Code, odd.Body.String())
	}

	long := doJSON(t, router, http.MethodPost, "/forum/posts", createForumPostRequest{Category: "health", Title: strings.Repeat("é", domain.MaxForumTitleLen+1), Body: "Texte"}, authorToken)
	if long.Code != http.StatusBadRequest || decodeJSON[errorResponse](t, long).Code != "forum.title_too_long" {
		t.Errorf("long title: status = %d, body = %s; want 400 forum.title_too_long", long.Code, long.Body.String())
	}

	listRec := doJSON(t, router, http.MethodGet, "/forum/posts?category=health", nil, "")
	if listRec.Code != http.StatusOK {
		t.Fatalf("list posts status = %d, want %d", listRec.Code, http.StatusOK)
	}
	found := false
	for _, p := range decodeJSON[[]forumPostResponse](t, listRec) {
		if p.ID == post.ID {
			found = true
		}
	}
	if !found {
		t.Error("new post doesn't appear in the category-filtered list")
	}

	replyRec := doJSON(t, router, http.MethodPost, "/forum/posts/"+post.ID+"/replies", addForumReplyRequest{
		Body: "Rappel à J35 en saison des pluies.",
	}, replierToken)
	if replyRec.Code != http.StatusCreated {
		t.Fatalf("add reply status = %d, want %d; body = %s", replyRec.Code, http.StatusCreated, replyRec.Body.String())
	}

	// Only the post's author can mark a best answer.
	forbidden := doJSON(t, router, http.MethodPost, "/forum/posts/"+post.ID+"/replies/0/mark-best", nil, replierToken)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("non-author mark-best status = %d, want %d", forbidden.Code, http.StatusForbidden)
	}

	markRec := doJSON(t, router, http.MethodPost, "/forum/posts/"+post.ID+"/replies/0/mark-best", nil, authorToken)
	if markRec.Code != http.StatusNoContent {
		t.Fatalf("mark-best status = %d, want %d; body = %s", markRec.Code, http.StatusNoContent, markRec.Body.String())
	}

	getRec := doJSON(t, router, http.MethodGet, "/forum/posts/"+post.ID, nil, "")
	got := decodeJSON[forumPostResponse](t, getRec)
	if len(got.Replies) != 1 {
		t.Fatalf("Replies = %d, want 1", len(got.Replies))
	}
	if got.BestReplyIndex == nil || *got.BestReplyIndex != 0 {
		t.Errorf("BestReplyIndex = %v, want pointer to 0", got.BestReplyIndex)
	}
}

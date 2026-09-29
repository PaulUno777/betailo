package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/forum"
	"github.com/pauluno777/betailo/backend/domain"
)

type forumReplyResponse struct {
	AuthorID  string    `json:"authorId"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

type forumPostResponse struct {
	ID             string               `json:"id"`
	AuthorID       string               `json:"authorId"`
	Category       string               `json:"category"`
	Title          string               `json:"title"`
	Body           string               `json:"body"`
	ListingID      *string              `json:"listingId,omitempty"`
	CreatedAt      time.Time            `json:"createdAt"`
	Replies        []forumReplyResponse `json:"replies,omitempty"`
	BestReplyIndex *int                 `json:"bestReplyIndex,omitempty"`
}

func toForumPostResponse(p *domain.ForumPost) forumPostResponse {
	resp := forumPostResponse{
		ID: p.ID.Hex(), AuthorID: p.AuthorID.Hex(), Category: p.Category, Title: p.Title,
		Body: p.Body, CreatedAt: p.CreatedAt, BestReplyIndex: p.BestReplyIndex,
	}
	if p.ListingID != nil {
		hex := p.ListingID.Hex()
		resp.ListingID = &hex
	}
	for _, rep := range p.Replies {
		resp.Replies = append(resp.Replies, forumReplyResponse{AuthorID: rep.AuthorID.Hex(), Body: rep.Body, CreatedAt: rep.CreatedAt})
	}
	return resp
}

type createForumPostRequest struct {
	Category  string `json:"category"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	ListingID string `json:"listingId,omitempty"`
}

func handleCreateForumPost(uc *forum.CreatePost) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		var req createForumPostRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, errBadRequestBody)
			return
		}
		var listingID *bson.ObjectID
		if req.ListingID != "" {
			id, err := parseObjectIDField(req.ListingID)
			if err != nil {
				writeError(w, errBadRequestBody)
				return
			}
			listingID = &id
		}

		p, err := uc.Execute(r.Context(), forum.CreatePostInput{
			AuthorID: claims.UserID, Category: req.Category, Title: req.Title, Body: req.Body, ListingID: listingID,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toForumPostResponse(p))
	}
}

func handleListForumPosts(uc *forum.ListPosts) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter := ports.ForumListFilter{
			Category: r.URL.Query().Get("category"),
			Query:    r.URL.Query().Get("q"),
			Sort:     r.URL.Query().Get("sort"),
		}
		posts, err := uc.Execute(r.Context(), filter)
		if err != nil {
			writeError(w, err)
			return
		}
		out := make([]forumPostResponse, 0, len(posts))
		for _, p := range posts {
			out = append(out, toForumPostResponse(p))
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleGetForumPost(uc *forum.GetPost) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		p, err := uc.Execute(r.Context(), id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toForumPostResponse(p))
	}
}

type addForumReplyRequest struct {
	Body string `json:"body"`
}

func handleAddForumReply(uc *forum.AddReply) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		postID, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		var req addForumReplyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, errBadRequestBody)
			return
		}
		if err := uc.Execute(r.Context(), forum.AddReplyInput{PostID: postID, AuthorID: claims.UserID, Body: req.Body}); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}
}

func handleMarkBestAnswer(uc *forum.MarkBestAnswer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		postID, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		index, err := strconv.Atoi(r.PathValue("index"))
		if err != nil {
			writeError(w, errMissingPathParam)
			return
		}
		if err := uc.Execute(r.Context(), forum.MarkBestAnswerInput{PostID: postID, AuthorID: claims.UserID, ReplyIndex: index}); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

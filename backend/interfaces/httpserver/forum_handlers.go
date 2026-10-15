package httpserver

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/forum"
	"github.com/pauluno777/betailo/backend/application/usecase/people"
	"github.com/pauluno777/betailo/backend/domain"
)

type forumReplyResponse struct {
	AuthorID string `json:"authorId"`
	// AuthorName is the author's public name, AuthorAvatar their profile
	// photo; both empty when unknown.
	AuthorName   string    `json:"authorName,omitempty"`
	AuthorAvatar string    `json:"authorAvatarUrl,omitempty"`
	Body         string    `json:"body"`
	CreatedAt    time.Time `json:"createdAt"`
}

type forumPostResponse struct {
	ID             string               `json:"id"`
	AuthorID       string               `json:"authorId"`
	AuthorName     string               `json:"authorName,omitempty"`
	AuthorAvatar   string               `json:"authorAvatarUrl,omitempty"`
	Category       string               `json:"category"`
	Title          string               `json:"title"`
	Body           string               `json:"body"`
	ListingID      *string              `json:"listingId,omitempty"`
	CreatedAt      time.Time            `json:"createdAt"`
	Replies        []forumReplyResponse `json:"replies,omitempty"`
	BestReplyIndex *int                 `json:"bestReplyIndex,omitempty"`
}

// forumView names the authors of posts and replies, each looked up once
// per response.
type forumView struct {
	People  *people.Directory
	Storage ports.ObjectStorage
}

func (v forumView) posts(ctx context.Context, posts []*domain.ForumPost) []forumPostResponse {
	var ids []bson.ObjectID
	for _, p := range posts {
		ids = append(ids, p.AuthorID)
		for _, rep := range p.Replies {
			ids = append(ids, rep.AuthorID)
		}
	}
	known := map[bson.ObjectID]people.Person{}
	if v.People != nil {
		known = v.People.Lookup(ctx, ids)
	}
	avatars := map[string]string{}
	avatar := func(key string) string {
		if url, ok := avatars[key]; ok {
			return url
		}
		avatars[key] = avatarURL(ctx, v.Storage, key)
		return avatars[key]
	}
	out := make([]forumPostResponse, 0, len(posts))
	for _, p := range posts {
		author := known[p.AuthorID]
		resp := forumPostResponse{
			ID: p.ID.Hex(), AuthorID: p.AuthorID.Hex(), AuthorName: author.Name, AuthorAvatar: avatar(author.AvatarKey),
			Category: p.Category, Title: p.Title, Body: p.Body, CreatedAt: p.CreatedAt, BestReplyIndex: p.BestReplyIndex,
		}
		if p.ListingID != nil {
			hex := p.ListingID.Hex()
			resp.ListingID = &hex
		}
		for _, rep := range p.Replies {
			who := known[rep.AuthorID]
			resp.Replies = append(resp.Replies, forumReplyResponse{
				AuthorID: rep.AuthorID.Hex(), AuthorName: who.Name, AuthorAvatar: avatar(who.AvatarKey), Body: rep.Body, CreatedAt: rep.CreatedAt,
			})
		}
		out = append(out, resp)
	}
	return out
}

func (v forumView) post(ctx context.Context, p *domain.ForumPost) forumPostResponse {
	return v.posts(ctx, []*domain.ForumPost{p})[0]
}

type createForumPostRequest struct {
	Category  string `json:"category"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	ListingID string `json:"listingId,omitempty"`
}

func handleCreateForumPost(uc *forum.CreatePost, view forumView) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		var req createForumPostRequest
		if !readJSON(w, r, &req) {
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
		writeJSON(w, http.StatusCreated, view.post(r.Context(), p))
	}
}

func handleListForumPosts(uc *forum.ListPosts, view forumView) http.HandlerFunc {
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
		writeJSON(w, http.StatusOK, view.posts(r.Context(), posts))
	}
}

func handleGetForumPost(uc *forum.GetPost, view forumView) http.HandlerFunc {
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
		writeJSON(w, http.StatusOK, view.post(r.Context(), p))
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
		if !readJSON(w, r, &req) {
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

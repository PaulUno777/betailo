package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/review"
	"github.com/pauluno777/betailo/backend/application/usecase/reviewview"
	"github.com/pauluno777/betailo/backend/domain"
)

type criteriaJSON struct {
	Health        int `json:"health,omitempty"`
	Conformity    int `json:"conformity,omitempty"`
	Communication int `json:"communication,omitempty"`
	Timeliness    int `json:"timeliness,omitempty"`
}

type reviewReplyResponse struct {
	Text string    `json:"text"`
	At   time.Time `json:"at"`
}

type reviewResponse struct {
	ID       string       `json:"id"`
	Rating   int          `json:"rating"`
	Comment  string       `json:"comment,omitempty"`
	Criteria criteriaJSON `json:"criteria"`
	Tags     []string     `json:"tags"`
	Photos   []string     `json:"photos"`
	// Thumbs are the photos' small copies, index for index (see
	// listingResponse.Thumbs).
	Thumbs         []string             `json:"thumbs"`
	HelpfulCount   int                  `json:"helpfulCount"`
	VotedByMe      bool                 `json:"votedByMe"`
	Mine           bool                 `json:"mine"`
	Reply          *reviewReplyResponse `json:"reply,omitempty"`
	AuthorName     string               `json:"authorName,omitempty"`
	ReservationRef string               `json:"reservationRef,omitempty"`
	CreatedAt      time.Time            `json:"createdAt"`
}

func toReviewResponse(ctx context.Context, storage ports.ObjectStorage, it reviewview.Item) reviewResponse {
	r := it.Review
	out := reviewResponse{
		ID: r.ID.Hex(), Rating: r.Rating, Comment: r.Comment,
		Criteria: criteriaJSON{Health: r.Criteria.Health, Conformity: r.Criteria.Conformity,
			Communication: r.Criteria.Communication, Timeliness: r.Criteria.Timeliness},
		Tags: r.Tags, Photos: []string{}, Thumbs: []string{}, HelpfulCount: r.HelpfulCount, VotedByMe: it.VotedByViewer, Mine: it.ByViewer,
		AuthorName: it.AuthorName, ReservationRef: it.ReservationRef, CreatedAt: r.CreatedAt,
	}
	if out.Tags == nil {
		out.Tags = []string{}
	}
	for _, key := range r.Photos {
		url, err := storage.PresignDownload(ctx, key)
		if err != nil {
			continue
		}
		thumb, err := storage.PresignDownload(ctx, domain.ThumbKey(key))
		if err != nil {
			thumb = url
		}
		out.Photos = append(out.Photos, url)
		out.Thumbs = append(out.Thumbs, thumb)
	}
	if r.Reply != nil {
		out.Reply = &reviewReplyResponse{Text: r.Reply.Text, At: r.Reply.At}
	}
	return out
}

type reviewSummaryResponse struct {
	Average   float64      `json:"average"`
	Count     int          `json:"count"`
	Histogram map[int]int  `json:"histogram"`
	Criteria  criteriaAvgs `json:"criteria"`
}

type criteriaAvgs struct {
	Health        float64 `json:"health"`
	Conformity    float64 `json:"conformity"`
	Communication float64 `json:"communication"`
	Timeliness    float64 `json:"timeliness"`
}

func toReviewSummaryResponse(s ports.ReviewSummary) reviewSummaryResponse {
	h := s.Histogram
	if h == nil {
		h = map[int]int{}
	}
	return reviewSummaryResponse{
		Average: s.Average, Count: s.Count, Histogram: h,
		Criteria: criteriaAvgs{Health: s.Criteria.Health, Conformity: s.Criteria.Conformity,
			Communication: s.Criteria.Communication, Timeliness: s.Criteria.Timeliness},
	}
}

type createReviewRequest struct {
	Rating    int          `json:"rating"`
	Comment   string       `json:"comment,omitempty"`
	Criteria  criteriaJSON `json:"criteria"`
	Tags      []string     `json:"tags,omitempty"`
	PhotoKeys []string     `json:"photoKeys,omitempty"`
}

func handleCreateReview(uc *review.CreateReview, storage ports.ObjectStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		reservationID, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		var req createReviewRequest
		if !readJSON(w, r, &req) {
			return
		}

		rv, err := uc.Execute(r.Context(), review.CreateReviewInput{
			ReservationID: reservationID, AuthorID: claims.UserID, Rating: req.Rating, Comment: req.Comment,
			Criteria: domain.ReviewCriteria{Health: req.Criteria.Health, Conformity: req.Criteria.Conformity,
				Communication: req.Criteria.Communication, Timeliness: req.Criteria.Timeliness},
			Tags: req.Tags, PhotoKeys: req.PhotoKeys,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toReviewResponse(r.Context(), storage, reviewview.Item{Review: rv}))
	}
}

func handlePresignReviewPhoto(uc *review.PresignReviewPhoto) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		reservationID, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		var req presignPhotoRequest
		if !readJSON(w, r, &req) {
			return
		}
		out, err := uc.Execute(r.Context(), reservationID, claims.UserID, req.Size, req.ThumbSize)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, presignPhotoResponse{Key: out.Key, UploadURL: out.UploadURL, ThumbUploadURL: out.ThumbUploadURL})
	}
}

type farmReviewsResponse struct {
	Items   []reviewResponse      `json:"items"`
	Total   int                   `json:"total"`
	Summary reviewSummaryResponse `json:"summary"`
}

func handleListFarmReviews(uc *review.ListByFarm, storage ports.ObjectStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		farmID, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		claims, _ := ClaimsFromContext(r.Context())
		q := r.URL.Query()
		out, err := uc.Execute(r.Context(), review.ListByFarmInput{
			FarmID: farmID, ViewerID: claims.UserID, Filter: ports.ReviewFilter(q.Get("filter")),
			Page: parseIntOr(q.Get("page"), 1), PageSize: parseIntOr(q.Get("pageSize"), 0),
		})
		if err != nil {
			writeError(w, err)
			return
		}
		items := make([]reviewResponse, 0, len(out.Items))
		for _, it := range out.Items {
			items = append(items, toReviewResponse(r.Context(), storage, it))
		}
		writeJSON(w, http.StatusOK, farmReviewsResponse{Items: items, Total: out.Total, Summary: toReviewSummaryResponse(out.Summary)})
	}
}

type helpfulResponse struct {
	HelpfulCount int `json:"helpfulCount"`
}

func handleMarkReviewHelpful(uc *review.MarkHelpful) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		n, err := uc.Execute(r.Context(), id, claims.UserID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, helpfulResponse{HelpfulCount: n})
	}
}

type reviewReplyRequest struct {
	Text string `json:"text"`
}

func handleReplyToReview(uc *review.Reply, storage ports.ObjectStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		var req reviewReplyRequest
		if !readJSON(w, r, &req) {
			return
		}
		rv, err := uc.Execute(r.Context(), review.ReplyInput{ReviewID: id, OwnerID: claims.UserID, Text: req.Text})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toReviewResponse(r.Context(), storage, reviewview.Item{Review: rv}))
	}
}

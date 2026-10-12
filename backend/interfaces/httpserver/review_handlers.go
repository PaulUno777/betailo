package httpserver

import (
	"net/http"

	"github.com/pauluno777/betailo/backend/application/usecase/review"
	"github.com/pauluno777/betailo/backend/domain"
)

type reviewResponse struct {
	ID      string `json:"id"`
	Rating  int    `json:"rating"`
	Comment string `json:"comment,omitempty"`
}

func toReviewResponse(r *domain.Review) reviewResponse {
	return reviewResponse{ID: r.ID.Hex(), Rating: r.Rating, Comment: r.Comment}
}

type reviewSummaryResponse struct {
	Average   float64     `json:"average"`
	Count     int         `json:"count"`
	Histogram map[int]int `json:"histogram"`
}

type createReviewRequest struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment,omitempty"`
}

func handleCreateReview(uc *review.CreateReview) http.HandlerFunc {
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
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toReviewResponse(rv))
	}
}

type farmReviewsResponse struct {
	Reviews []reviewResponse      `json:"reviews"`
	Summary reviewSummaryResponse `json:"summary"`
}

func handleListFarmReviews(uc *review.ListByFarm) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		farmID, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		out, err := uc.Execute(r.Context(), farmID)
		if err != nil {
			writeError(w, err)
			return
		}
		reviews := make([]reviewResponse, 0, len(out.Reviews))
		for _, rv := range out.Reviews {
			reviews = append(reviews, toReviewResponse(rv))
		}
		writeJSON(w, http.StatusOK, farmReviewsResponse{
			Reviews: reviews,
			Summary: reviewSummaryResponse{Average: out.Summary.Average, Count: out.Summary.Count, Histogram: out.Summary.Histogram},
		})
	}
}

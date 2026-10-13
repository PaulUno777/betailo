package httpserver

import (
	"context"
	"net/http"
	"slices"
	"time"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/reservation"
	"github.com/pauluno777/betailo/backend/domain"
)

// paymentInfoResponse is a Reservation's embedded escrow sub-document
// over the wire.
type paymentInfoResponse struct {
	Status     string     `json:"status"`
	Amount     float64    `json:"amount"`
	HeldAt     *time.Time `json:"heldAt,omitempty"`
	ReleasedAt *time.Time `json:"releasedAt,omitempty"`

	Method           string `json:"method,omitempty"`
	PayerPhoneMasked string `json:"payerPhoneMasked,omitempty"`
}

// reservationResponse is what a reservation looks like over the wire —
// every field here is state the backend already decided (fee, priority,
// deadline, escrow status); the client only ever renders it.
type reservationResponse struct {
	ID          string               `json:"id"`
	Reference   string               `json:"reference"`
	ListingID   string               `json:"listingId"`
	BuyerID     string               `json:"buyerId"`
	Quantity    int                  `json:"quantity"`
	UnitPrice   float64              `json:"unitPrice"`
	ServiceFee  float64              `json:"serviceFee"`
	TotalAmount float64              `json:"totalAmount"`
	Status      string               `json:"status"`
	Priority    bool                 `json:"priority"`
	DeliveredAt *time.Time           `json:"deliveredAt,omitempty"`
	CreatedAt   time.Time            `json:"createdAt"`
	Payment     *paymentInfoResponse `json:"payment,omitempty"`
}

func toReservationResponse(r *domain.Reservation) reservationResponse {
	resp := reservationResponse{
		ID: r.ID.Hex(), Reference: r.Reference, ListingID: r.ListingID.Hex(), BuyerID: r.BuyerID.Hex(),
		Quantity: r.Quantity, UnitPrice: r.UnitPrice, ServiceFee: r.ServiceFee, TotalAmount: r.TotalAmount,
		Status: string(r.Status), Priority: r.Priority,
		DeliveredAt: r.DeliveredAt, CreatedAt: r.CreatedAt,
	}
	if r.Payment != nil {
		resp.Payment = &paymentInfoResponse{
			Status: string(r.Payment.Status), Amount: r.Payment.Amount,
			HeldAt: r.Payment.HeldAt, ReleasedAt: r.Payment.ReleasedAt,
			Method: r.Payment.Method, PayerPhoneMasked: r.Payment.PayerPhoneMasked,
		}
	}
	return resp
}

type createReservationRequest struct {
	Quantity      int    `json:"quantity"`
	PayNow        bool   `json:"payNow"`
	PaymentMethod string `json:"paymentMethod,omitempty"`
	PayerPhone    string `json:"payerPhone,omitempty"`
}

type createReservationResponse struct {
	Reservation     reservationResponse `json:"reservation"`
	PaymentCaptured bool                `json:"paymentCaptured"`
}

func handleCreateReservation(uc *reservation.ReserveCycle) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		listingID, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		var req createReservationRequest
		if !readJSON(w, r, &req) {
			return
		}

		out, err := uc.Execute(r.Context(), reservation.ReserveCycleInput{
			ListingID: listingID, BuyerID: claims.UserID, Quantity: req.Quantity,
			PayNow: req.PayNow, PaymentMethod: ports.PaymentMethod(req.PaymentMethod), PayerPhone: req.PayerPhone,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, createReservationResponse{
			Reservation: toReservationResponse(out.Reservation), PaymentCaptured: out.PaymentCaptured,
		})
	}
}

type payReservationRequest struct {
	PaymentMethod string `json:"paymentMethod,omitempty"`
	PayerPhone    string `json:"payerPhone,omitempty"`
}

func handlePayReservation(uc *reservation.PayReservation) http.HandlerFunc {
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
		var req payReservationRequest
		if !readJSON(w, r, &req) {
			return
		}

		res, err := uc.Execute(r.Context(), reservation.PayReservationInput{
			ReservationID: id, BuyerID: claims.UserID, PaymentMethod: ports.PaymentMethod(req.PaymentMethod), PayerPhone: req.PayerPhone,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toReservationResponse(res))
	}
}

func handleCancelReservation(uc *reservation.CancelReservation) http.HandlerFunc {
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
		res, err := uc.Execute(r.Context(), reservation.CancelReservationInput{ReservationID: id, BuyerID: claims.UserID})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toReservationResponse(res))
	}
}

func handleMarkDelivered(uc *reservation.MarkDelivered) http.HandlerFunc {
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
		res, err := uc.Execute(r.Context(), reservation.MarkDeliveredInput{ReservationID: id, FarmOwnerID: claims.UserID})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toReservationResponse(res))
	}
}

func handleConfirmDelivery(uc *reservation.ConfirmDelivery) http.HandlerFunc {
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
		res, err := uc.Execute(r.Context(), reservation.ConfirmDeliveryInput{ReservationID: id, BuyerID: claims.UserID})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toReservationResponse(res))
	}
}

type openDisputeRequest struct {
	Reason string `json:"reason"`
}

func handleOpenDispute(uc *reservation.OpenDispute) http.HandlerFunc {
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
		var req openDisputeRequest
		if !readJSON(w, r, &req) {
			return
		}

		if _, err := uc.Execute(r.Context(), reservation.OpenDisputeInput{
			ReservationID: id, BuyerID: claims.UserID, Reason: req.Reason,
		}); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

// reservationViewResponse is a reservation card: the reservation, what
// was reserved and from whom, and the server's decisions about its tab,
// progress and next actions (see reservation.View).
type reservationViewResponse struct {
	reservationResponse
	Listing struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Unit  string `json:"unit,omitempty"`
		Photo string `json:"photo,omitempty"`
	} `json:"listing"`
	Farm struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Location     string `json:"location"`
		ContactPhone string `json:"contactPhone,omitempty"`
	} `json:"farm"`
	Cycle struct {
		ID        string     `json:"id"`
		Reference string     `json:"reference"`
		Status    string     `json:"status"`
		StartsAt  time.Time  `json:"startsAt"`
		EndsAt    *time.Time `json:"endsAt,omitempty"`
	} `json:"cycle"`
	Group         string               `json:"group"`
	Steps         []escrowStepResponse `json:"steps"`
	AutoReleaseAt *time.Time           `json:"autoReleaseAt,omitempty"`
	DisputeOpen   bool                 `json:"disputeOpen"`
	Reviewed      bool                 `json:"reviewed"`
	Actions       []string             `json:"actions"`
}

type escrowStepResponse struct {
	Key   string `json:"key"`
	State string `json:"state"`
}

type reservationListResponse struct {
	Items       []reservationViewResponse `json:"items"`
	Counts      map[string]int            `json:"counts"`
	EscrowTotal float64                   `json:"escrowTotal"`
}

func toReservationListResponse(ctx context.Context, storage ports.ObjectStorage, list *reservation.ViewList) reservationListResponse {
	out := reservationListResponse{Items: make([]reservationViewResponse, 0, len(list.Items)), Counts: map[string]int{}, EscrowTotal: list.EscrowTotal}
	for g, n := range list.Counts {
		out.Counts[string(g)] = n
	}
	for _, v := range list.Items {
		item := reservationViewResponse{reservationResponse: toReservationResponse(v.Reservation)}
		item.Listing.ID, item.Listing.Title, item.Listing.Unit = v.Listing.ID.Hex(), v.Listing.Title, v.Listing.Unit
		if len(v.Listing.Photos) > 0 {
			if url, err := storage.PresignDownload(ctx, v.Listing.Photos[0]); err == nil {
				item.Listing.Photo = url
			}
		}
		item.Farm.ID, item.Farm.Name, item.Farm.Location = v.Farm.ID.Hex(), v.Farm.Name, v.Farm.Location
		if slices.Contains(v.Actions, reservation.ActionContact) {
			item.Farm.ContactPhone = v.Farm.ContactPhone
		}
		item.Cycle.ID, item.Cycle.Reference, item.Cycle.Status = v.Cycle.ID.Hex(), v.Cycle.Reference, string(v.Cycle.Status)
		item.Cycle.StartsAt, item.Cycle.EndsAt = v.Cycle.StartsAt, v.Cycle.EndsAt
		item.Group, item.AutoReleaseAt, item.DisputeOpen, item.Reviewed = string(v.Group), v.AutoReleaseAt, v.DisputeOpen, v.Reviewed
		item.Steps = make([]escrowStepResponse, 0, len(v.Steps))
		for _, st := range v.Steps {
			item.Steps = append(item.Steps, escrowStepResponse{Key: string(st.Key), State: string(st.State)})
		}
		item.Actions = make([]string, 0, len(v.Actions))
		for _, a := range v.Actions {
			item.Actions = append(item.Actions, string(a))
		}
		out.Items = append(out.Items, item)
	}
	return out
}

func handleListMyReservations(uc *reservation.ListMyReservations, storage ports.ObjectStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		list, err := uc.Execute(r.Context(), claims.UserID, domain.ReservationGroup(r.URL.Query().Get("group")))
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toReservationListResponse(r.Context(), storage, list))
	}
}

func handleListReservationsByFarm(uc *reservation.ListReservationsByFarm, storage ports.ObjectStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		farmID, ok := pathObjectID(w, r, "farmId")
		if !ok {
			return
		}
		list, err := uc.Execute(r.Context(), reservation.ListReservationsByFarmInput{
			FarmID: farmID, OwnerID: claims.UserID, Group: domain.ReservationGroup(r.URL.Query().Get("group")),
			Query: r.URL.Query().Get("q"),
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toReservationListResponse(r.Context(), storage, list))
	}
}

type quoteRequest struct {
	Quantity int  `json:"quantity"`
	PayNow   bool `json:"payNow"`
}

type quoteResponse struct {
	Quantity   int     `json:"quantity"`
	UnitPrice  float64 `json:"unitPrice"`
	Subtotal   float64 `json:"subtotal"`
	ServiceFee float64 `json:"serviceFee"`
	Total      float64 `json:"total"`
	FeeRate    float64 `json:"feeRate"`
	Min        int     `json:"min"`
	Max        int     `json:"max"`
	Available  int     `json:"available"`
}

func handleQuoteReservation(uc *reservation.Quote) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		listingID, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		var req quoteRequest
		if !readJSON(w, r, &req) {
			return
		}
		q, err := uc.Execute(r.Context(), reservation.QuoteInput{ListingID: listingID, BuyerID: claims.UserID, Quantity: req.Quantity, PayNow: req.PayNow})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, quoteResponse{
			Quantity: q.Quantity, UnitPrice: q.UnitPrice, Subtotal: q.Subtotal, ServiceFee: q.ServiceFee, Total: q.Total,
			FeeRate: q.FeeRate, Min: q.Min, Max: q.Max, Available: q.Available,
		})
	}
}

// navCountsResponse is the navigation's badge numbers.
type navCountsResponse struct {
	Buyer  int `json:"buyer"`
	Seller int `json:"seller"`
	// Cycles is how many of the seller's cycles are planned or running.
	Cycles int `json:"cycles"`
}

func handleNavCounts(uc *reservation.NavCounts) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		out, err := uc.Execute(r.Context(), claims.UserID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, navCountsResponse{Buyer: out.Buyer, Seller: out.Seller, Cycles: out.Cycles})
	}
}

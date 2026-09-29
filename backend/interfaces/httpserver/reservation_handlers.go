package httpserver

import (
	"encoding/json"
	"net/http"
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
}

// reservationResponse is what a reservation looks like over the wire —
// every field here is state the backend already decided (fee, priority,
// deadline, escrow status); the client only ever renders it.
type reservationResponse struct {
	ID              string               `json:"id"`
	ListingID       string               `json:"listingId"`
	BuyerID         string               `json:"buyerId"`
	Quantity        int                  `json:"quantity"`
	UnitPrice       float64              `json:"unitPrice"`
	ServiceFee      float64              `json:"serviceFee"`
	TotalAmount     float64              `json:"totalAmount"`
	Status          string               `json:"status"`
	Priority        bool                 `json:"priority"`
	PaymentDeadline *time.Time           `json:"paymentDeadline,omitempty"`
	DeliveredAt     *time.Time           `json:"deliveredAt,omitempty"`
	CreatedAt       time.Time            `json:"createdAt"`
	Payment         *paymentInfoResponse `json:"payment,omitempty"`
}

func toReservationResponse(r *domain.Reservation) reservationResponse {
	resp := reservationResponse{
		ID: r.ID.Hex(), ListingID: r.ListingID.Hex(), BuyerID: r.BuyerID.Hex(),
		Quantity: r.Quantity, UnitPrice: r.UnitPrice, ServiceFee: r.ServiceFee, TotalAmount: r.TotalAmount,
		Status: string(r.Status), Priority: r.Priority, PaymentDeadline: r.PaymentDeadline,
		DeliveredAt: r.DeliveredAt, CreatedAt: r.CreatedAt,
	}
	if r.Payment != nil {
		resp.Payment = &paymentInfoResponse{
			Status: string(r.Payment.Status), Amount: r.Payment.Amount,
			HeldAt: r.Payment.HeldAt, ReleasedAt: r.Payment.ReleasedAt,
		}
	}
	return resp
}

type createReservationRequest struct {
	Quantity      int    `json:"quantity"`
	PayNow        bool   `json:"payNow"`
	PaymentMethod string `json:"paymentMethod,omitempty"`
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
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, errBadRequestBody)
			return
		}

		out, err := uc.Execute(r.Context(), reservation.ReserveCycleInput{
			ListingID: listingID, BuyerID: claims.UserID, Quantity: req.Quantity,
			PayNow: req.PayNow, PaymentMethod: ports.PaymentMethod(req.PaymentMethod),
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
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, errBadRequestBody)
			return
		}

		res, err := uc.Execute(r.Context(), reservation.PayReservationInput{
			ReservationID: id, BuyerID: claims.UserID, PaymentMethod: ports.PaymentMethod(req.PaymentMethod),
		})
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
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, errBadRequestBody)
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

func handleListMyReservations(uc *reservation.ListMyReservations) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, errUnauthorized)
			return
		}
		reservations, err := uc.Execute(r.Context(), claims.UserID)
		if err != nil {
			writeError(w, err)
			return
		}
		out := make([]reservationResponse, 0, len(reservations))
		for _, res := range reservations {
			out = append(out, toReservationResponse(res))
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleListReservationsByFarm(uc *reservation.ListReservationsByFarm) http.HandlerFunc {
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
		reservations, err := uc.Execute(r.Context(), reservation.ListReservationsByFarmInput{
			FarmID: farmID, OwnerID: claims.UserID,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		out := make([]reservationResponse, 0, len(reservations))
		for _, res := range reservations {
			out = append(out, toReservationResponse(res))
		}
		writeJSON(w, http.StatusOK, out)
	}
}

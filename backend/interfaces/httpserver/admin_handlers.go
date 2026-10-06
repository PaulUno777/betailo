package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/admin"
	"github.com/pauluno777/betailo/backend/application/usecase/reservation"
	"github.com/pauluno777/betailo/backend/domain"
)

type overviewResponse struct {
	VolumeReserved         float64 `json:"volumeReserved"`
	EscrowHeld             float64 `json:"escrowHeld"`
	ActiveCyclesPlanned    int     `json:"activeCyclesPlanned"`
	ActiveCyclesInProgress int     `json:"activeCyclesInProgress"`
	OpenReports            int     `json:"openReports"`
	ReservationsPerDay     [30]int `json:"reservationsPerDay"`
}

func handleAdminOverview(uc *admin.Overview) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := uc.Execute(r.Context())
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, overviewResponse{
			VolumeReserved: out.VolumeReserved, EscrowHeld: out.EscrowHeld,
			ActiveCyclesPlanned: out.ActiveCyclesPlanned, ActiveCyclesInProgress: out.ActiveCyclesInProgress,
			OpenReports: out.OpenReports, ReservationsPerDay: out.ReservationsPerDay,
		})
	}
}

type adminUserResponse struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	Role        string  `json:"role"`
	DisplayName string  `json:"displayName,omitempty"`
	IsVerified  bool    `json:"isVerified"`
	Suspended   bool    `json:"suspended"`
	TrustScore  float64 `json:"trustScore"`
}

func toAdminUserResponse(u *domain.User) adminUserResponse {
	return adminUserResponse{
		ID: u.ID.Hex(), Email: u.Email, Role: string(u.Role), DisplayName: u.DisplayName,
		IsVerified: u.IsVerified, Suspended: u.Suspended, TrustScore: u.TrustScore,
	}
}

func handleAdminListUsers(uc *admin.ListUsers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := uc.Execute(r.Context())
		if err != nil {
			writeError(w, err)
			return
		}
		out := make([]adminUserResponse, 0, len(users))
		for _, u := range users {
			out = append(out, toAdminUserResponse(u))
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleAdminVerifyUser(uc *admin.VerifyUser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		u, err := uc.Execute(r.Context(), id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toAdminUserResponse(u))
	}
}

func handleAdminSuspendUser(uc *admin.SuspendUser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		u, err := uc.Execute(r.Context(), id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toAdminUserResponse(u))
	}
}

func handleAdminListCycles(uc *admin.ListAllCycles) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cycles, err := uc.Execute(r.Context())
		if err != nil {
			writeError(w, err)
			return
		}
		out := make([]cycleResponse, 0, len(cycles))
		for _, c := range cycles {
			out = append(out, toCycleResponse(c))
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleAdminListListings(uc *admin.ListAllListings, storage ports.ObjectStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		listings, err := uc.Execute(r.Context())
		if err != nil {
			writeError(w, err)
			return
		}
		out := make([]listingResponse, 0, len(listings))
		for _, l := range listings {
			out = append(out, toListingResponse(r.Context(), storage, l))
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleAdminListEscrow(uc *admin.ListEscrowTransactions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reservations, err := uc.Execute(r.Context())
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

func handleAdminReleaseEscrow(uc *reservation.AdminReleaseEscrow) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		res, err := uc.Execute(r.Context(), id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toReservationResponse(res))
	}
}

func handleAdminRefundEscrow(uc *reservation.AdminRefundEscrow) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		res, err := uc.Execute(r.Context(), id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toReservationResponse(res))
	}
}

func handleAdminListReports(uc *admin.ListReports) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := domain.ReportStatus(r.URL.Query().Get("status"))
		reports, err := uc.Execute(r.Context(), status)
		if err != nil {
			writeError(w, err)
			return
		}
		out := make([]reportResponse, 0, len(reports))
		for _, rp := range reports {
			out = append(out, toReportResponse(rp))
		}
		writeJSON(w, http.StatusOK, out)
	}
}

type riskSignalResponse struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
	Weight int    `json:"weight"`
}

type reportDetailResponse struct {
	Report    reportResponse       `json:"report"`
	History   []reportResponse     `json:"history"`
	RiskScore int                  `json:"riskScore"`
	Signals   []riskSignalResponse `json:"signals"`
}

func handleAdminGetReportDetail(uc *admin.GetReportDetail) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		detail, err := uc.Execute(r.Context(), id)
		if err != nil {
			writeError(w, err)
			return
		}
		history := make([]reportResponse, 0, len(detail.History))
		for _, rp := range detail.History {
			history = append(history, toReportResponse(rp))
		}
		signals := make([]riskSignalResponse, 0, len(detail.Signals))
		for _, s := range detail.Signals {
			signals = append(signals, riskSignalResponse{Name: s.Name, Detail: s.Detail, Weight: s.Weight})
		}
		writeJSON(w, http.StatusOK, reportDetailResponse{
			Report: toReportResponse(detail.Report), History: history, RiskScore: detail.RiskScore, Signals: signals,
		})
	}
}

type resolveReportRequest struct {
	Action string `json:"action"`
}

func handleAdminResolveReport(uc *admin.ResolveReport) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		var req resolveReportRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, errBadRequestBody)
			return
		}
		rp, err := uc.Execute(r.Context(), admin.ResolveReportInput{ReportID: id, Action: admin.ResolveAction(req.Action)})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toReportResponse(rp))
	}
}

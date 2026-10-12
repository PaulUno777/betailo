package httpserver

import (
	"net/http"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/admin"
	"github.com/pauluno777/betailo/backend/application/usecase/farm"
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
	ID          string `json:"id"`
	Email       string `json:"email"`
	StaffRole   string `json:"staffRole,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	// FarmLimit is what staff set for this account; 0 = platform default.
	FarmLimit  int     `json:"farmLimit"`
	Suspended  bool    `json:"suspended"`
	TrustScore float64 `json:"trustScore"`
}

func toAdminUserResponse(u *domain.User) adminUserResponse {
	return adminUserResponse{
		ID: u.ID.Hex(), Email: u.Email, StaffRole: string(u.StaffRole), DisplayName: u.DisplayName,
		FarmLimit: u.FarmLimit, Suspended: u.Suspended, TrustScore: u.TrustScore,
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

type farmLimitRequest struct {
	Limit int `json:"limit"`
}

func handleAdminSetFarmLimit(uc *admin.SetFarmLimit) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		var req farmLimitRequest
		if !readJSON(w, r, &req) {
			return
		}
		u, err := uc.Execute(r.Context(), id, req.Limit)
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
		if !readJSON(w, r, &req) {
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

// adminFarmResponse is a farm in the validation queue, with its owner.
type adminFarmResponse struct {
	farmResponse
	Owner *adminUserResponse `json:"owner,omitempty"`
}

func toAdminFarmResponse(item admin.FarmReviewItem) adminFarmResponse {
	out := adminFarmResponse{farmResponse: toOwnerFarmResponse(item.Farm)}
	if item.Owner != nil {
		owner := toAdminUserResponse(item.Owner)
		out.Owner = &owner
	}
	return out
}

func handleAdminListFarms(uc *admin.ListFarmsForReview) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := uc.Execute(r.Context(), domain.FarmStatus(r.URL.Query().Get("status")))
		if err != nil {
			writeError(w, err)
			return
		}
		out := make([]adminFarmResponse, 0, len(items))
		for _, item := range items {
			out = append(out, toAdminFarmResponse(item))
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// adminFarmDetailResponse adds the proof (short-lived links) staff
// decide from.
type adminFarmDetailResponse struct {
	adminFarmResponse
	IDDocumentURL  string   `json:"idDocumentUrl,omitempty"`
	PhotoURLs      []string `json:"photoUrls"`
	OwnerFarmCount int      `json:"ownerFarmCount"`
}

func handleAdminGetFarm(uc *admin.GetFarmForReview) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		d, err := uc.Execute(r.Context(), id)
		if err != nil {
			writeError(w, err)
			return
		}
		photos := d.PhotoURLs
		if photos == nil {
			photos = []string{}
		}
		writeJSON(w, http.StatusOK, adminFarmDetailResponse{
			adminFarmResponse: toAdminFarmResponse(d.FarmReviewItem), IDDocumentURL: d.IDDocumentURL, PhotoURLs: photos, OwnerFarmCount: d.OwnerFarmCount,
		})
	}
}

type reviewFarmRequest struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason,omitempty"`
}

func handleAdminReviewFarm(uc *farm.ReviewFarm) http.HandlerFunc {
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
		var req reviewFarmRequest
		if !readJSON(w, r, &req) {
			return
		}
		f, err := uc.Execute(r.Context(), farm.ReviewFarmInput{FarmID: id, ReviewerID: claims.UserID, Decision: farm.FarmDecision(req.Decision), Reason: req.Reason})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toOwnerFarmResponse(f))
	}
}

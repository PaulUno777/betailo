package httpserver

import (
	"net/http"
	"strconv"
	"time"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/admin"
	"github.com/pauluno777/betailo/backend/application/usecase/farm"
	"github.com/pauluno777/betailo/backend/application/usecase/reservation"
	"github.com/pauluno777/betailo/backend/domain"
)

// staffActor reads the staff member RequirePermission let through,
// answering 403 itself when there is none.
func staffActor(w http.ResponseWriter, r *http.Request) (admin.Actor, bool) {
	actor, ok := actorFromContext(r.Context())
	if !ok {
		writeError(w, errForbidden)
	}
	return actor, ok
}

type overviewBucketResponse struct {
	Start   time.Time `json:"start"`
	Paid    int       `json:"paid"`
	Waiting int       `json:"waiting"`
}

type overviewResponse struct {
	Period                 string                   `json:"period"`
	VolumeReserved         float64                  `json:"volumeReserved"`
	VolumePrevious         float64                  `json:"volumePrevious"`
	EscrowHeld             float64                  `json:"escrowHeld"`
	HeldReservations       int                      `json:"heldReservations"`
	ActiveCyclesPlanned    int                      `json:"activeCyclesPlanned"`
	ActiveCyclesInProgress int                      `json:"activeCyclesInProgress"`
	OpenReports            int                      `json:"openReports"`
	OpenDisputes           int                      `json:"openDisputes"`
	PendingFarms           int                      `json:"pendingFarms"`
	OldestPendingSince     *time.Time               `json:"oldestPendingSince,omitempty"`
	Series                 []overviewBucketResponse `json:"series"`
	// AutoApproveFarms and MaxFarmsPerAccount are the deployment's
	// switches, read-only here.
	AutoApproveFarms   bool `json:"autoApproveFarms"`
	MaxFarmsPerAccount int  `json:"maxFarmsPerAccount"`
}

func handleAdminOverview(uc *admin.Overview) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := uc.Execute(r.Context(), admin.Period(r.URL.Query().Get("period")))
		if err != nil {
			writeError(w, err)
			return
		}
		series := make([]overviewBucketResponse, 0, len(out.Series))
		for _, b := range out.Series {
			series = append(series, overviewBucketResponse{Start: b.Start, Paid: b.Paid, Waiting: b.Waiting})
		}
		writeJSON(w, http.StatusOK, overviewResponse{
			Period: string(out.Period), VolumeReserved: out.VolumeReserved, VolumePrevious: out.VolumePrevious,
			EscrowHeld: out.EscrowHeld, HeldReservations: out.HeldReservations,
			ActiveCyclesPlanned: out.ActiveCyclesPlanned, ActiveCyclesInProgress: out.ActiveCyclesInProgress,
			OpenReports: out.OpenReports, OpenDisputes: out.OpenDisputes,
			PendingFarms: out.PendingFarms, OldestPendingSince: out.OldestPendingSince, Series: series,
			AutoApproveFarms: out.AutoApproveFarms, MaxFarmsPerAccount: out.MaxFarms,
		})
	}
}

type adminNavCountsResponse struct {
	PendingFarms int `json:"pendingFarms"`
	OpenReports  int `json:"openReports"`
	OpenDisputes int `json:"openDisputes"`
}

func handleAdminNavCounts(uc *admin.CountNav) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := uc.Execute(r.Context())
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, adminNavCountsResponse{PendingFarms: out.PendingFarms, OpenReports: out.OpenReports, OpenDisputes: out.OpenDisputes})
	}
}

type farmLimitNoteResponse struct {
	Reason string    `json:"reason"`
	SetAt  time.Time `json:"setAt"`
	// SetByName and SetByRole are only filled where the caller needs
	// them (the exceptions list).
	SetByName string `json:"setByName,omitempty"`
	SetByRole string `json:"setByRole,omitempty"`
}

type adminUserResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	StaffRole   string `json:"staffRole,omitempty"`
	IsRoot      bool   `json:"isRoot,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	// FarmLimit is what staff set for this account; 0 = platform default.
	FarmLimit     int                    `json:"farmLimit"`
	FarmLimitNote *farmLimitNoteResponse `json:"farmLimitNote,omitempty"`
	FarmCount     int                    `json:"farmCount"`
	Suspended     bool                   `json:"suspended"`
	TrustScore    float64                `json:"trustScore"`
	MemberSince   time.Time              `json:"memberSince"`
}

func toAdminUserResponse(u *domain.User, farmCount int) adminUserResponse {
	out := adminUserResponse{
		ID: u.ID.Hex(), Email: u.Email, StaffRole: string(u.StaffRole), IsRoot: u.Root, DisplayName: u.DisplayName,
		FarmLimit: u.FarmLimit, FarmCount: farmCount, Suspended: u.Suspended, TrustScore: u.TrustScore, MemberSince: u.CreatedAt,
	}
	if n := u.FarmLimitNote; n != nil {
		out.FarmLimitNote = &farmLimitNoteResponse{Reason: n.Reason, SetAt: n.SetAt}
	}
	return out
}

func handleAdminListUsers(uc *admin.ListUsers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := uc.Execute(r.Context())
		if err != nil {
			writeError(w, err)
			return
		}
		out := make([]adminUserResponse, 0, len(items))
		for _, item := range items {
			out = append(out, toAdminUserResponse(item.User, item.FarmCount))
		}
		writeJSON(w, http.StatusOK, out)
	}
}

type farmLimitRequest struct {
	// Email names the account on the exceptions form; the per-account
	// route takes it from the path instead.
	Email  string `json:"email,omitempty"`
	Limit  int    `json:"limit"`
	Reason string `json:"reason"`
}

// handleAdminSetFarmLimit serves both `POST /admin/users/{id}/farm-limit`
// and `POST /admin/farm-limits` (account named by email).
func handleAdminSetFarmLimit(uc *admin.SetFarmLimit, journal *admin.Journal) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := staffActor(w, r)
		if !ok {
			return
		}
		in := admin.SetFarmLimitInput{Actor: actor}
		if r.PathValue("id") != "" {
			id, ok := pathObjectID(w, r, "id")
			if !ok {
				return
			}
			in.UserID = id
		}
		var req farmLimitRequest
		if !readJSON(w, r, &req) {
			return
		}
		in.Email, in.Limit, in.Reason = req.Email, req.Limit, req.Reason
		item, err := uc.Execute(r.Context(), in)
		if err != nil {
			writeError(w, err)
			return
		}
		detail := strconv.Itoa(req.Limit)
		if req.Reason != "" {
			detail += " · " + req.Reason
		}
		journal.Record(r.Context(), actor, admin.Entry{
			Action: domain.ActionFarmLimitSet, TargetType: "user", TargetID: item.User.ID, TargetLabel: item.User.Email, Detail: detail,
		})
		writeJSON(w, http.StatusOK, toAdminUserResponse(item.User, item.FarmCount))
	}
}

type farmLimitsResponse struct {
	// Default is MAX_FARMS_PER_ACCOUNT: set by the deployment, not here.
	Default    int                 `json:"default"`
	Max        int                 `json:"max"`
	Exceptions []adminUserResponse `json:"exceptions"`
}

func handleAdminListFarmLimits(uc *admin.ListFarmLimits) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := uc.Execute(r.Context())
		if err != nil {
			writeError(w, err)
			return
		}
		exceptions := make([]adminUserResponse, 0, len(out.Exceptions))
		for _, ex := range out.Exceptions {
			item := toAdminUserResponse(ex.User, ex.FarmCount)
			if item.FarmLimitNote != nil {
				item.FarmLimitNote.SetByName, item.FarmLimitNote.SetByRole = ex.SetByName, string(ex.SetByRole)
			}
			exceptions = append(exceptions, item)
		}
		writeJSON(w, http.StatusOK, farmLimitsResponse{Default: out.Default, Max: out.Max, Exceptions: exceptions})
	}
}

func handleAdminSuspendUser(uc *admin.SuspendUser, journal *admin.Journal) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := staffActor(w, r)
		if !ok {
			return
		}
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		u, err := uc.Execute(r.Context(), actor, id)
		if err != nil {
			writeError(w, err)
			return
		}
		journal.Record(r.Context(), actor, admin.Entry{Action: domain.ActionUserSuspend, TargetType: "user", TargetID: u.ID, TargetLabel: u.Email})
		writeJSON(w, http.StatusOK, toAdminUserResponse(u, 0))
	}
}

func handleAdminReactivateUser(uc *admin.ReactivateUser, journal *admin.Journal) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := staffActor(w, r)
		if !ok {
			return
		}
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		u, err := uc.Execute(r.Context(), actor, id)
		if err != nil {
			writeError(w, err)
			return
		}
		journal.Record(r.Context(), actor, admin.Entry{Action: domain.ActionUserReactivate, TargetType: "user", TargetID: u.ID, TargetLabel: u.Email})
		writeJSON(w, http.StatusOK, toAdminUserResponse(u, 0))
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

// adminEscrowResponse is a reservation in the "Séquestre" table, with
// the buyer and the farm named.
type adminEscrowResponse struct {
	reservationResponse
	BuyerName string `json:"buyerName,omitempty"`
	FarmName  string `json:"farmName,omitempty"`
}

func handleAdminListEscrow(uc *admin.ListEscrowTransactions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := uc.Execute(r.Context())
		if err != nil {
			writeError(w, err)
			return
		}
		out := make([]adminEscrowResponse, 0, len(items))
		for _, item := range items {
			out = append(out, adminEscrowResponse{reservationResponse: toReservationResponse(item.Reservation), BuyerName: item.BuyerName, FarmName: item.FarmName})
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleAdminReleaseEscrow(uc *reservation.AdminReleaseEscrow, journal *admin.Journal) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := staffActor(w, r)
		if !ok {
			return
		}
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		res, err := uc.Execute(r.Context(), id)
		if err != nil {
			writeError(w, err)
			return
		}
		journal.Record(r.Context(), actor, admin.Entry{
			Action: domain.ActionEscrowRelease, TargetType: "reservation", TargetID: res.ID, TargetLabel: res.Reference,
			Detail: strconv.FormatFloat(res.TotalAmount, 'f', 0, 64) + " FCFA",
		})
		writeJSON(w, http.StatusOK, toReservationResponse(res))
	}
}

func handleAdminRefundEscrow(uc *reservation.AdminRefundEscrow, journal *admin.Journal) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := staffActor(w, r)
		if !ok {
			return
		}
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		res, err := uc.Execute(r.Context(), id)
		if err != nil {
			writeError(w, err)
			return
		}
		journal.Record(r.Context(), actor, admin.Entry{
			Action: domain.ActionEscrowRefund, TargetType: "reservation", TargetID: res.ID, TargetLabel: res.Reference,
			Detail: strconv.FormatFloat(res.TotalAmount, 'f', 0, 64) + " FCFA",
		})
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

func handleAdminResolveReport(uc *admin.ResolveReport, journal *admin.Journal) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := staffActor(w, r)
		if !ok {
			return
		}
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		var req resolveReportRequest
		if !readJSON(w, r, &req) {
			return
		}
		rp, err := uc.Execute(r.Context(), admin.ResolveReportInput{Actor: actor, ReportID: id, Action: admin.ResolveAction(req.Action)})
		if err != nil {
			writeError(w, err)
			return
		}
		journal.Record(r.Context(), actor, admin.Entry{
			Action: domain.ActionReportResolve, TargetType: "report", TargetID: rp.ID, TargetLabel: string(rp.TargetType), Detail: req.Action,
		})
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
		owner := toAdminUserResponse(item.Owner, item.OwnerFarmCount)
		out.Owner = &owner
	}
	return out
}

type adminFarmListResponse struct {
	Farms []adminFarmResponse `json:"farms"`
	// Counts are the sizes of the tabs still to work through.
	Counts adminFarmCountsResponse `json:"counts"`
}

type adminFarmCountsResponse struct {
	Todo        int `json:"todo"`
	Resubmitted int `json:"resubmitted"`
}

// handleAdminListFarms answers `GET /admin/farms?queue=todo|resubmitted|done`
// (the validation page's tabs) or `?status=<farm status>`.
func handleAdminListFarms(uc *admin.ListFarmsForReview, counts *admin.CountFarmQueues) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		items, err := uc.Execute(r.Context(), admin.ListFarmsInput{Queue: admin.FarmQueue(q.Get("queue")), Status: domain.FarmStatus(q.Get("status"))})
		if err != nil {
			writeError(w, err)
			return
		}
		sizes, err := counts.Execute(r.Context())
		if err != nil {
			writeError(w, err)
			return
		}
		out := make([]adminFarmResponse, 0, len(items))
		for _, item := range items {
			out = append(out, toAdminFarmResponse(item))
		}
		writeJSON(w, http.StatusOK, adminFarmListResponse{Farms: out, Counts: adminFarmCountsResponse{Todo: sizes.Todo, Resubmitted: sizes.Resubmitted}})
	}
}

// adminFarmDetailResponse adds the proof (short-lived links) staff
// decide from, and what the owner's account did so far.
type adminFarmDetailResponse struct {
	adminFarmResponse
	IDDocumentURL     string   `json:"idDocumentUrl,omitempty"`
	IDDocumentBackURL string   `json:"idDocumentBackUrl,omitempty"`
	PhotoURLs         []string `json:"photoUrls"`
	OwnerFarmCount    int      `json:"ownerFarmCount"`
	OwnerFarmLimit    int      `json:"ownerFarmLimit"`
	OwnerPurchases    int      `json:"ownerPurchases"`
	OwnerReports      int      `json:"ownerReports"`
	ReviewerName      string   `json:"reviewerName,omitempty"`
	ReviewerRole      string   `json:"reviewerRole,omitempty"`
	// RejectionCodes are the refusal reasons staff pick from.
	RejectionCodes []string `json:"rejectionCodes"`
}

// handleAdminGetFarm hands out the proof, and writes that it did: who
// looked at an identity document is part of the journal.
func handleAdminGetFarm(uc *admin.GetFarmForReview, journal *admin.Journal) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := staffActor(w, r)
		if !ok {
			return
		}
		id, ok := pathObjectID(w, r, "id")
		if !ok {
			return
		}
		d, err := uc.Execute(r.Context(), id)
		if err != nil {
			writeError(w, err)
			return
		}
		if d.IDDocumentURL != "" {
			journal.Record(r.Context(), actor, admin.Entry{
				Action: domain.ActionFarmProofViewed, TargetType: "farm", TargetID: d.Farm.ID, TargetLabel: farmLabel(d.Farm),
			})
		}
		photos := d.PhotoURLs
		if photos == nil {
			photos = []string{}
		}
		codes := make([]string, 0, len(domain.RejectionCodes))
		for _, c := range domain.RejectionCodes {
			codes = append(codes, string(c))
		}
		writeJSON(w, http.StatusOK, adminFarmDetailResponse{
			adminFarmResponse: toAdminFarmResponse(d.FarmReviewItem),
			IDDocumentURL:     d.IDDocumentURL, IDDocumentBackURL: d.IDDocumentBackURL, PhotoURLs: photos,
			OwnerFarmCount: d.OwnerFarmCount, OwnerFarmLimit: d.OwnerFarmLimit, OwnerPurchases: d.OwnerPurchases, OwnerReports: d.OwnerReports,
			ReviewerName: d.ReviewerName, ReviewerRole: string(d.ReviewerRole), RejectionCodes: codes,
		})
	}
}

// farmLabel is what the journal calls a farm.
func farmLabel(f *domain.Farm) string {
	if f.Reference != "" {
		return f.Reference + " · " + f.Name
	}
	return f.Name
}

type reviewFarmRequest struct {
	Decision string `json:"decision"`
	// ReasonCode is the kind of refusal, Reason the message to the owner.
	ReasonCode string `json:"reasonCode,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

func handleAdminReviewFarm(uc *farm.ReviewFarm, journal *admin.Journal) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := staffActor(w, r)
		if !ok {
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
		f, err := uc.Execute(r.Context(), farm.ReviewFarmInput{
			FarmID: id, ReviewerID: actor.ID, Decision: farm.FarmDecision(req.Decision),
			ReasonCode: domain.RejectionCode(req.ReasonCode), Reason: req.Reason,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		action := map[farm.FarmDecision]string{
			farm.FarmDecisionApprove: domain.ActionFarmApprove, farm.FarmDecisionReject: domain.ActionFarmReject, farm.FarmDecisionSuspend: domain.ActionFarmSuspend,
		}[farm.FarmDecision(req.Decision)]
		detail := f.Verification.RejectionReason
		if code := f.Verification.RejectionCode; code != "" {
			detail = string(code) + " · " + detail
		}
		journal.Record(r.Context(), actor, admin.Entry{Action: action, TargetType: "farm", TargetID: f.ID, TargetLabel: farmLabel(f), Detail: detail})
		writeJSON(w, http.StatusOK, toOwnerFarmResponse(f))
	}
}

type staffRoleResponse struct {
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

type staffResponse struct {
	Members []adminUserResponse `json:"members"`
	// Roles is the permission table as the API enforces it, and
	// Permissions every permission in display order: the client draws
	// the matrix from these, it doesn't keep its own copy.
	Roles       []staffRoleResponse `json:"roles"`
	Permissions []string            `json:"permissions"`
}

func handleAdminListStaff(uc *admin.ListStaff) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		members, err := uc.Execute(r.Context())
		if err != nil {
			writeError(w, err)
			return
		}
		out := staffResponse{Members: make([]adminUserResponse, 0, len(members))}
		for _, m := range members {
			out.Members = append(out.Members, toAdminUserResponse(m, 0))
		}
		for _, role := range domain.StaffRoles {
			perms := []string{}
			for _, p := range role.Permissions() {
				perms = append(perms, string(p))
			}
			out.Roles = append(out.Roles, staffRoleResponse{Role: string(role), Permissions: perms})
		}
		for _, p := range domain.Permissions {
			out.Permissions = append(out.Permissions, string(p))
		}
		writeJSON(w, http.StatusOK, out)
	}
}

type staffRoleRequest struct {
	// Email names the account when granting a first role; the
	// per-member route takes it from the path instead.
	Email string `json:"email,omitempty"`
	// Role is one of domain.StaffRoles; empty revokes.
	Role string `json:"role"`
}

// handleAdminSetStaffRole serves both `POST /admin/staff` (account
// named by email) and `POST /admin/staff/{id}/role`.
func handleAdminSetStaffRole(uc *admin.SetStaffRole, journal *admin.Journal) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := staffActor(w, r)
		if !ok {
			return
		}
		in := admin.SetStaffRoleInput{Actor: actor}
		if r.PathValue("id") != "" {
			id, ok := pathObjectID(w, r, "id")
			if !ok {
				return
			}
			in.UserID = id
		}
		var req staffRoleRequest
		if !readJSON(w, r, &req) {
			return
		}
		in.Email, in.Role = req.Email, domain.StaffRole(req.Role)
		if in.UserID.IsZero() && in.Role == domain.StaffNone {
			writeError(w, domain.Coded("admin.staff_role_unknown", errBadRequestBody))
			return
		}
		out, err := uc.Execute(r.Context(), in)
		if err != nil {
			writeError(w, err)
			return
		}
		if out.Previous != out.User.StaffRole {
			action, detail := domain.ActionStaffGrant, string(out.User.StaffRole)
			if !out.User.StaffRole.IsStaff() {
				action, detail = domain.ActionStaffRevoke, string(out.Previous)
			} else if out.Previous.IsStaff() {
				detail = string(out.Previous) + " → " + detail
			}
			journal.Record(r.Context(), actor, admin.Entry{Action: action, TargetType: "user", TargetID: out.User.ID, TargetLabel: out.User.Email, Detail: detail})
		}
		writeJSON(w, http.StatusOK, toAdminUserResponse(out.User, 0))
	}
}

type adminActionResponse struct {
	ID          string    `json:"id"`
	ActorName   string    `json:"actorName,omitempty"`
	ActorRole   string    `json:"actorRole,omitempty"`
	Action      string    `json:"action"`
	TargetType  string    `json:"targetType,omitempty"`
	TargetID    string    `json:"targetId,omitempty"`
	TargetLabel string    `json:"targetLabel,omitempty"`
	Detail      string    `json:"detail,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

func handleAdminListAudit(uc *admin.ListAudit) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actions, err := uc.Execute(r.Context())
		if err != nil {
			writeError(w, err)
			return
		}
		out := make([]adminActionResponse, 0, len(actions))
		for _, a := range actions {
			item := adminActionResponse{
				ID: a.ID.Hex(), ActorName: a.ActorName, ActorRole: string(a.ActorRole), Action: a.Action,
				TargetType: a.TargetType, TargetLabel: a.TargetLabel, Detail: a.Detail, CreatedAt: a.CreatedAt,
			}
			if !a.TargetID.IsZero() {
				item.TargetID = a.TargetID.Hex()
			}
			out = append(out, item)
		}
		writeJSON(w, http.StatusOK, out)
	}
}

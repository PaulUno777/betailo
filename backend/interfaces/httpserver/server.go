// Package httpserver is the delivery layer shared by every business
// module: the router, cross-cutting middlewares (logging, auth) and
// generic JSON helpers.
//
// Business handlers will register their routes on the *http.ServeMux
// built here — this package never contains business logic itself, and
// depends on application/usecase types, never on infrastructure/mongodb
// or infrastructure/objectstorage directly (dependencies flow inward:
// delivery -> application -> domain, never the reverse).
package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/pauluno777/betailo/backend/domain"
)

// chain applies middlewares to h in the order given — chain(h, A, B)
// means A wraps B wraps h, so A runs first. Only exists to keep the
// route table below readable; it has no behavior of its own.
func chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// NewRouter builds the application's top-level router.
//
// Creating a farm, cycle or listing is restricted to farmers
// (RequireRole) — everything else that mutates one of those (update,
// delete, a cycle's status) is protected by RequireAuth alone and
// relies on the usecase's own ownership check instead: a buyer could
// never own a farm to begin with, so a second role gate there would be
// redundant. Reads (GET) need no authentication — farms, cycles and
// published listings are the public marketplace.
func NewRouter(logger *slog.Logger, deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	auth := RequireAuth(deps.Signer)
	farmerOnly := func(h http.Handler) http.Handler { return chain(h, auth, RequireRole(domain.RoleFarmer)) }

	mux.HandleFunc("GET /health", handleHealth)
	limit := func(policy RateLimit, h http.Handler) http.Handler {
		return rateLimited(policy, deps.RateLimits.TrustProxyHeaders, time.Now, h)
	}
	mux.Handle("POST /auth/register", limit(deps.RateLimits.Register, handleRegister(deps.Register)))
	mux.Handle("POST /auth/login", limit(deps.RateLimits.Login, handleLogin(deps.Login, deps.Cookies)))
	// Neither reads Authorization — each authenticates via the refresh
	// cookie itself, which matters for refresh specifically: that's the
	// one call still expected to work after the access token has
	// already expired.
	mux.Handle("POST /auth/refresh", limit(deps.RateLimits.Refresh, handleRefresh(deps.RefreshToken, deps.Cookies)))
	mux.HandleFunc("POST /auth/logout", handleLogout(deps.Logout, deps.Cookies))
	mux.Handle("GET /me", chain(http.HandlerFunc(handleMe), auth))

	mux.Handle("POST /farms", farmerOnly(handleCreateFarm(deps.CreateFarm)))
	mux.HandleFunc("GET /farms/{id}", handleGetFarm(deps.GetFarm))
	mux.Handle("GET /farms/mine", chain(handleListMyFarms(deps.ListMyFarms), auth))
	mux.Handle("PUT /farms/{id}", chain(handleUpdateFarm(deps.UpdateFarm), auth))
	mux.Handle("DELETE /farms/{id}", chain(handleDeleteFarm(deps.DeleteFarm), auth))

	mux.Handle("POST /cycles", farmerOnly(handleCreateCycle(deps.CreateCycle)))
	mux.HandleFunc("GET /cycles/{id}", handleGetCycle(deps.GetCycle))
	mux.HandleFunc("GET /farms/{farmId}/cycles", handleListCyclesByFarm(deps.ListCyclesByFarm))
	mux.Handle("PATCH /cycles/{id}/status", chain(handleUpdateCycleStatus(deps.UpdateCycleStatus), auth))
	mux.Handle("DELETE /cycles/{id}", chain(handleDeleteCycle(deps.DeleteCycle), auth))

	mux.Handle("POST /listings", farmerOnly(handleCreateListing(deps.CreateListing, deps.Storage)))
	mux.HandleFunc("GET /listings/{id}", handleGetListing(deps.GetListing, deps.Storage))
	mux.HandleFunc("GET /listings", handleSearchListings(deps.SearchListings, deps.Storage))
	mux.Handle("PUT /listings/{id}", chain(handleUpdateListing(deps.UpdateListing, deps.Storage), auth))
	mux.Handle("DELETE /listings/{id}", chain(handleUnpublishListing(deps.UnpublishListing), auth))
	mux.Handle("POST /listings/{id}/photos/presign", chain(handlePresignListingPhoto(deps.PresignListingPhoto), auth))
	mux.Handle("PUT /listings/{id}/photos", chain(handleSetListingPhotos(deps.SetListingPhotos, deps.Storage), auth))

	// Reservation & escrow. Creating one needs no role restriction — a
	// farmer can buy too — but does need auth, since BuyerID/FarmOwnerID
	// always come from the token, never a request body field.
	mux.Handle("POST /listings/{id}/reservations", chain(handleCreateReservation(deps.ReserveCycle), auth))
	mux.Handle("GET /reservations/mine", chain(handleListMyReservations(deps.ListMyReservations), auth))
	mux.Handle("POST /reservations/{id}/pay", chain(handlePayReservation(deps.PayReservation), auth))
	mux.Handle("POST /reservations/{id}/mark-delivered", chain(handleMarkDelivered(deps.MarkDelivered), auth))
	mux.Handle("POST /reservations/{id}/confirm-delivery", chain(handleConfirmDelivery(deps.ConfirmDelivery), auth))
	mux.Handle("POST /reservations/{id}/dispute", chain(handleOpenDispute(deps.OpenDispute), auth))
	mux.Handle("GET /farms/{farmId}/reservations", chain(handleListReservationsByFarm(deps.ListReservationsByFarm), auth))

	// Reviews — writing one needs auth (the buyer only); reading a
	// farm's reviews is public, like its listings.
	mux.Handle("POST /reservations/{id}/reviews", chain(handleCreateReview(deps.CreateReview), auth))
	mux.HandleFunc("GET /farms/{id}/reviews", handleListFarmReviews(deps.ListFarmReviews))

	// Reporting — any authenticated role can report.
	mux.Handle("POST /reports", chain(handleCreateReport(deps.CreateReport), auth))

	// Forum — reading is public; posting/replying needs auth.
	mux.Handle("POST /forum/posts", chain(handleCreateForumPost(deps.CreateForumPost), auth))
	mux.HandleFunc("GET /forum/posts", handleListForumPosts(deps.ListForumPosts))
	mux.HandleFunc("GET /forum/posts/{id}", handleGetForumPost(deps.GetForumPost))
	mux.Handle("POST /forum/posts/{id}/replies", chain(handleAddForumReply(deps.AddForumReply), auth))
	mux.Handle("POST /forum/posts/{id}/replies/{index}/mark-best", chain(handleMarkBestAnswer(deps.MarkBestAnswer), auth))

	// Seller (farmer) portal — ownership is checked inside the usecase
	// itself, the same as every other /farms/{farmId}/... route.
	mux.Handle("GET /farms/{farmId}/dashboard", chain(handleSellerDashboard(deps.SellerDashboard), auth))

	// Admin (root) console — every route here is gated on the admin
	// role alone; each usecase reads/writes platform-wide, with no
	// further ownership check of its own (see application/usecase/admin's
	// package doc).
	adminOnly := func(h http.Handler) http.Handler { return chain(h, auth, RequireRole(domain.RoleAdmin)) }
	mux.Handle("GET /admin/overview", adminOnly(handleAdminOverview(deps.AdminOverview)))
	mux.Handle("GET /admin/users", adminOnly(handleAdminListUsers(deps.AdminListUsers)))
	mux.Handle("POST /admin/users/{id}/verify", adminOnly(handleAdminVerifyUser(deps.AdminVerifyUser)))
	mux.Handle("POST /admin/users/{id}/suspend", adminOnly(handleAdminSuspendUser(deps.AdminSuspendUser)))
	mux.Handle("GET /admin/cycles", adminOnly(handleAdminListCycles(deps.AdminListCycles)))
	mux.Handle("GET /admin/listings", adminOnly(handleAdminListListings(deps.AdminListListings, deps.Storage)))
	mux.Handle("GET /admin/escrow", adminOnly(handleAdminListEscrow(deps.AdminListEscrow)))
	mux.Handle("POST /admin/escrow/{id}/release", adminOnly(handleAdminReleaseEscrow(deps.AdminReleaseEscrow)))
	mux.Handle("POST /admin/escrow/{id}/refund", adminOnly(handleAdminRefundEscrow(deps.AdminRefundEscrow)))
	mux.Handle("GET /admin/reports", adminOnly(handleAdminListReports(deps.AdminListReports)))
	mux.Handle("GET /admin/reports/{id}", adminOnly(handleAdminGetReportDetail(deps.AdminGetReportDetail)))
	mux.Handle("POST /admin/reports/{id}/resolve", adminOnly(handleAdminResolveReport(deps.AdminResolveReport)))

	return withCORS(withLogging(logger, mux))
}

type healthResponse struct {
	Status string `json:"status"`
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

// writeJSON is a small shared helper — every handler in every module
// should respond through it (or an equivalent) so error shapes stay
// consistent across the API.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// withLogging logs one line per request: method, path, status and
// duration. Kept intentionally simple for the MVP — swap for a richer
// middleware (request id, tracing) if the project grows past portfolio
// scope.
func withLogging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		logger.Info("request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rec.status),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

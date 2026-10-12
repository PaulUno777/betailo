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
// There is one kind of account. Anything that creates or changes a
// farm, cycle or listing is protected by RequireAuth and by the
// usecase's own ownership check: you sell from a farm you own, and
// only an approved farm's listings go live (see domain.FarmStatus).
// Reads (GET) need no authentication — farms, cycles and
// published listings are the public marketplace.
func NewRouter(logger *slog.Logger, deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	auth := RequireAuth(deps.Signer, deps.Accounts)
	signedIn := func(h http.Handler) http.Handler { return chain(h, auth) }

	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /reference/geo", handleGeoReference)
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
	mux.Handle("GET /me", chain(handleGetProfile(deps.GetProfile, deps.SellerAccess), auth))
	mux.Handle("PATCH /me", chain(handleUpdateProfile(deps.UpdateProfile), auth))
	// Guessing the current password is rate-limited like login.
	mux.Handle("POST /me/password", limit(deps.RateLimits.Login, chain(handleChangePassword(deps.ChangePassword, deps.Login, deps.Cookies), auth)))

	mux.Handle("POST /farms", signedIn(handleCreateFarm(deps.CreateFarm)))
	mux.Handle("GET /farms/{id}", chain(handleGetFarmProfile(deps.GetFarmProfile), OptionalAuth(deps.Signer, deps.Accounts)))
	mux.Handle("GET /farms/mine", chain(handleListMyFarms(deps.ListMyFarms), auth))
	mux.Handle("PUT /farms/{id}", chain(handleUpdateFarm(deps.UpdateFarm), auth))
	mux.Handle("DELETE /farms/{id}", chain(handleDeleteFarm(deps.DeleteFarm), auth))
	mux.Handle("POST /farms/{id}/verification/presign", signedIn(handlePresignFarmProof(deps.PresignFarmProof)))
	mux.Handle("POST /farms/{id}/verification/submit", signedIn(handleSubmitFarmForReview(deps.SubmitFarmForReview)))

	mux.Handle("POST /cycles", signedIn(handleCreateCycle(deps.CreateCycle)))
	mux.HandleFunc("GET /cycles/{id}", handleGetCycle(deps.GetCycle))
	mux.HandleFunc("GET /farms/{farmId}/cycles", handleListCyclesByFarm(deps.ListCyclesByFarm))
	mux.Handle("PATCH /cycles/{id}/status", chain(handleUpdateCycleStatus(deps.UpdateCycleStatus), auth))
	mux.Handle("DELETE /cycles/{id}", chain(handleDeleteCycle(deps.DeleteCycle), auth))
	mux.Handle("PUT /cycles/{id}", chain(handleUpdateCycle(deps.UpdateCycle), auth))
	mux.Handle("POST /cycles/{id}/health-events", chain(handleAddHealthEvent(deps.AddHealthEvent), auth))

	mux.Handle("POST /listings", signedIn(handleCreateListing(deps.CreateListing, deps.Storage)))
	mux.Handle("GET /listings/{id}", chain(handleGetListing(deps.GetListing, deps.Storage), OptionalAuth(deps.Signer, deps.Accounts)))
	mux.Handle("GET /listings/{id}/detail", chain(handleGetListingDetail(deps.GetListingDetail, deps.Storage), OptionalAuth(deps.Signer, deps.Accounts)))
	mux.Handle("GET /listings", chain(handleSearchListings(deps.SearchListings, deps.Storage), OptionalAuth(deps.Signer, deps.Accounts)))
	mux.Handle("POST /farms/{farmId}/quick-listings", signedIn(handleQuickCreateListing(deps.QuickCreateListing, deps.Storage)))
	mux.Handle("POST /listings/{id}/publish", chain(handlePublishListing(deps.PublishListing, deps.Storage), auth))
	mux.Handle("GET /farms/{farmId}/listings", chain(handleListFarmListings(deps.ListFarmListings, deps.Storage), auth))
	mux.HandleFunc("GET /listings/price-hint", handlePriceHint(deps.PriceHint))
	mux.Handle("PUT /listings/{id}/favorite", chain(handleAddFavorite(deps.AddFavorite), auth))
	mux.Handle("DELETE /listings/{id}/favorite", chain(handleRemoveFavorite(deps.RemoveFavorite), auth))
	mux.Handle("GET /notifications", chain(handleListNotifications(deps.ListNotifications), auth))
	mux.Handle("GET /notifications/unread-count", chain(handleUnreadNotifications(deps.UnreadNotifications), auth))
	mux.Handle("POST /notifications/{id}/read", chain(handleMarkNotificationRead(deps.MarkNotificationRead), auth))
	mux.Handle("POST /notifications/read-all", chain(handleMarkAllNotificationsRead(deps.MarkAllNotificationsRead), auth))
	mux.Handle("GET /me/favorites", chain(handleListMyFavorites(deps.ListMyFavorites, deps.Storage), auth))
	mux.Handle("PUT /listings/{id}", chain(handleUpdateListing(deps.UpdateListing, deps.Storage), auth))
	mux.Handle("DELETE /listings/{id}", chain(handleUnpublishListing(deps.UnpublishListing), auth))
	mux.Handle("POST /listings/{id}/photos/presign", chain(handlePresignListingPhoto(deps.PresignListingPhoto), auth))
	mux.Handle("PUT /listings/{id}/photos", chain(handleSetListingPhotos(deps.SetListingPhotos, deps.Storage), auth))

	// Reservation & escrow. Creating one needs no role restriction — a
	// farmer can buy too — but does need auth, since BuyerID/FarmOwnerID
	// always come from the token, never a request body field.
	mux.Handle("POST /listings/{id}/quote", chain(handleQuoteReservation(deps.QuoteReservation), auth))
	mux.Handle("POST /listings/{id}/reservations", chain(handleCreateReservation(deps.ReserveCycle), auth))
	mux.Handle("GET /me/nav-counts", chain(handleNavCounts(deps.NavCounts), auth))
	mux.Handle("GET /reservations/mine", chain(handleListMyReservations(deps.ListMyReservations, deps.Storage), auth))
	mux.Handle("POST /reservations/{id}/pay", chain(handlePayReservation(deps.PayReservation), auth))
	mux.Handle("POST /reservations/{id}/mark-delivered", chain(handleMarkDelivered(deps.MarkDelivered), auth))
	mux.Handle("POST /reservations/{id}/confirm-delivery", chain(handleConfirmDelivery(deps.ConfirmDelivery), auth))
	mux.Handle("POST /reservations/{id}/cancel", chain(handleCancelReservation(deps.CancelReservation), auth))
	mux.Handle("POST /reservations/{id}/dispute", chain(handleOpenDispute(deps.OpenDispute), auth))
	mux.Handle("GET /farms/{farmId}/reservations", chain(handleListReservationsByFarm(deps.ListReservationsByFarm, deps.Storage), auth))

	// Reviews — writing one needs auth (the buyer only); reading a
	// farm's reviews is public, like its listings.
	mux.Handle("POST /reservations/{id}/reviews", chain(handleCreateReview(deps.CreateReview, deps.Storage), auth))
	mux.Handle("POST /reservations/{id}/reviews/photos/presign", chain(handlePresignReviewPhoto(deps.PresignReviewPhoto), auth))
	mux.Handle("GET /farms/{id}/reviews", chain(handleListFarmReviews(deps.ListFarmReviews, deps.Storage), OptionalAuth(deps.Signer, deps.Accounts)))
	mux.Handle("POST /reviews/{id}/helpful", chain(handleMarkReviewHelpful(deps.MarkReviewHelpful), auth))
	mux.Handle("POST /reviews/{id}/reply", chain(handleReplyToReview(deps.ReplyToReview, deps.Storage), auth))

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

	// Back office — one permission per route (see domain.Permission for
	// which staff role holds what). A read that every role needs to
	// draw its own navigation asks for the overview permission, which
	// they all hold.
	can := func(p domain.Permission, h http.Handler) http.Handler {
		return chain(h, auth, RequirePermission(deps.StaffRoles, p))
	}
	journal := deps.AdminJournal
	mux.Handle("GET /admin/overview", can(domain.PermOverviewRead, handleAdminOverview(deps.AdminOverview)))
	mux.Handle("GET /admin/nav-counts", can(domain.PermOverviewRead, handleAdminNavCounts(deps.AdminNavCounts)))

	mux.Handle("GET /admin/users", can(domain.PermAccountsRead, handleAdminListUsers(deps.AdminListUsers)))
	mux.Handle("POST /admin/users/{id}/suspend", can(domain.PermAccountsSuspend, handleAdminSuspendUser(deps.AdminSuspendUser, journal)))
	mux.Handle("POST /admin/users/{id}/reactivate", can(domain.PermAccountsSuspend, handleAdminReactivateUser(deps.AdminReactivateUser, journal)))
	mux.Handle("POST /admin/users/{id}/farm-limit", can(domain.PermFarmLimitSet, handleAdminSetFarmLimit(deps.AdminSetFarmLimit, journal)))
	mux.Handle("GET /admin/farm-limits", can(domain.PermAccountsRead, handleAdminListFarmLimits(deps.AdminListFarmLimits)))
	mux.Handle("POST /admin/farm-limits", can(domain.PermFarmLimitSet, handleAdminSetFarmLimit(deps.AdminSetFarmLimit, journal)))

	mux.Handle("GET /admin/farms", can(domain.PermFarmsRead, handleAdminListFarms(deps.AdminListFarms, deps.AdminFarmQueueCounts)))
	mux.Handle("GET /admin/farms/{id}", can(domain.PermFarmsRead, handleAdminGetFarm(deps.AdminGetFarm, journal)))
	mux.Handle("POST /admin/farms/{id}/review", can(domain.PermFarmsReview, handleAdminReviewFarm(deps.ReviewFarm, journal)))

	mux.Handle("GET /admin/cycles", can(domain.PermCatalogRead, handleAdminListCycles(deps.AdminListCycles)))
	mux.Handle("GET /admin/listings", can(domain.PermCatalogRead, handleAdminListListings(deps.AdminListListings, deps.Storage)))

	mux.Handle("GET /admin/escrow", can(domain.PermEscrowRead, handleAdminListEscrow(deps.AdminListEscrow)))
	mux.Handle("POST /admin/escrow/{id}/release", can(domain.PermEscrowSettle, handleAdminReleaseEscrow(deps.AdminReleaseEscrow, journal)))
	mux.Handle("POST /admin/escrow/{id}/refund", can(domain.PermEscrowSettle, handleAdminRefundEscrow(deps.AdminRefundEscrow, journal)))

	mux.Handle("GET /admin/reports", can(domain.PermReportsRead, handleAdminListReports(deps.AdminListReports)))
	mux.Handle("GET /admin/reports/{id}", can(domain.PermReportsRead, handleAdminGetReportDetail(deps.AdminGetReportDetail)))
	mux.Handle("POST /admin/reports/{id}/resolve", can(domain.PermReportsResolve, handleAdminResolveReport(deps.AdminResolveReport, journal)))

	// The team itself. Reading who is on it and what each role may do
	// is open to all staff (it is how a refusal gets explained);
	// changing it is the super admin's alone.
	mux.Handle("GET /admin/staff", can(domain.PermOverviewRead, handleAdminListStaff(deps.AdminListStaff)))
	mux.Handle("POST /admin/staff", can(domain.PermStaffManage, handleAdminSetStaffRole(deps.AdminSetStaffRole, journal)))
	mux.Handle("POST /admin/staff/{id}/role", can(domain.PermStaffManage, handleAdminSetStaffRole(deps.AdminSetStaffRole, journal)))
	mux.Handle("GET /admin/audit", can(domain.PermAuditRead, handleAdminListAudit(deps.AdminListAudit)))

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

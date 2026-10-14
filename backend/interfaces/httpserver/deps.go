package httpserver

import (
	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/account"
	"github.com/pauluno777/betailo/backend/application/usecase/admin"
	"github.com/pauluno777/betailo/backend/application/usecase/auth"
	"github.com/pauluno777/betailo/backend/application/usecase/cycle"
	"github.com/pauluno777/betailo/backend/application/usecase/dashboard"
	"github.com/pauluno777/betailo/backend/application/usecase/farm"
	"github.com/pauluno777/betailo/backend/application/usecase/favorite"
	"github.com/pauluno777/betailo/backend/application/usecase/forum"
	"github.com/pauluno777/betailo/backend/application/usecase/inbox"
	"github.com/pauluno777/betailo/backend/application/usecase/listing"
	"github.com/pauluno777/betailo/backend/application/usecase/report"
	"github.com/pauluno777/betailo/backend/application/usecase/reservation"
	"github.com/pauluno777/betailo/backend/application/usecase/review"
)

// Dependencies is everything NewRouter needs to wire routes to
// usecases. A struct, not a growing parameter list, so adding the next
// business module's usecase (see the roadmap) doesn't change every
// existing call to NewRouter.
type Dependencies struct {
	Register     *auth.Register
	Login        *auth.Login
	RefreshToken *auth.RefreshToken
	Logout       *auth.Logout
	Signer       ports.TokenSigner
	// Accounts ends the sessions of suspended accounts (nil: no check).
	Accounts AccountChecker
	Cookies  CookieConfig

	GetProfile     *account.GetProfile
	UpdateProfile  *account.UpdateProfile
	ChangePassword *auth.ChangePassword
	SellerAccess   *farm.SellerAccess
	// RateLimits applies to the credential endpoints; the zero value
	// disables limiting.
	RateLimits RateLimitConfig

	ListNotifications        *inbox.List
	UnreadNotifications      *inbox.UnreadCount
	MarkNotificationRead     *inbox.MarkRead
	MarkAllNotificationsRead *inbox.MarkAllRead

	AddFavorite     *favorite.Add
	RemoveFavorite  *favorite.Remove
	ListMyFavorites *favorite.ListMine

	CreateFarm     *farm.CreateFarm
	GetFarmProfile *farm.GetProfile
	ListMyFarms    *farm.ListMyFarms
	UpdateFarm     *farm.UpdateFarm
	DeleteFarm     *farm.DeleteFarm

	PresignFarmProof    *farm.PresignFarmProof
	SubmitFarmForReview *farm.SubmitFarmForReview
	ReviewFarm          *farm.ReviewFarm

	CreateCycle       *cycle.CreateCycle
	GetCycle          *cycle.GetCycle
	ListCyclesByFarm  *cycle.ListCyclesByFarm
	UpdateCycleStatus *cycle.UpdateCycleStatus
	AddHealthEvent    *cycle.AddHealthEvent
	UpdateCycle       *cycle.UpdateCycle
	DeleteCycle       *cycle.DeleteCycle

	CreateListing       *listing.CreateListing
	GetListing          *listing.GetListing
	GetListingDetail    *listing.GetDetail
	QuickCreateListing  *listing.QuickCreate
	PublishListing      *listing.Publish
	ListFarmListings    *listing.ListFarmListings
	PriceHint           *listing.PriceHint
	UpdateListing       *listing.UpdateListing
	UnpublishListing    *listing.UnpublishListing
	PresignListingPhoto *listing.PresignListingPhoto
	SetListingPhotos    *listing.SetListingPhotos
	SearchListings      *listing.SearchListings
	Storage             ports.ObjectStorage

	CancelReservation      *reservation.CancelReservation
	QuoteReservation       *reservation.Quote
	ReserveCycle           *reservation.ReserveCycle
	PayReservation         *reservation.PayReservation
	MarkDelivered          *reservation.MarkDelivered
	ConfirmDelivery        *reservation.ConfirmDelivery
	OpenDispute            *reservation.OpenDispute
	ListMyReservations     *reservation.ListMyReservations
	ListReservationsByFarm *reservation.ListReservationsByFarm
	NavCounts              *reservation.NavCounts

	CreateReview       *review.CreateReview
	ListFarmReviews    *review.ListByFarm
	PresignReviewPhoto *review.PresignReviewPhoto
	MarkReviewHelpful  *review.MarkHelpful
	ReplyToReview      *review.Reply

	CreateReport *report.CreateReport

	CreateForumPost *forum.CreatePost
	ListForumPosts  *forum.ListPosts
	GetForumPost    *forum.GetPost
	AddForumReply   *forum.AddReply
	MarkBestAnswer  *forum.MarkBestAnswer

	SellerDashboard *dashboard.SellerDashboard

	AdminOverview        *admin.Overview
	AdminListUsers       *admin.ListUsers
	AdminSetFarmLimit    *admin.SetFarmLimit
	AdminListFarms       *admin.ListFarmsForReview
	AdminGetFarm         *admin.GetFarmForReview
	AdminSuspendUser     *admin.SuspendUser
	AdminListCycles      *admin.ListAllCycles
	AdminListListings    *admin.ListAllListings
	AdminListEscrow      *admin.ListEscrowTransactions
	AdminReleaseEscrow   *reservation.AdminReleaseEscrow
	AdminRefundEscrow    *reservation.AdminRefundEscrow
	AdminListReports     *admin.ListReports
	AdminGetReportDetail *admin.GetReportDetail
	AdminResolveReport   *admin.ResolveReport
}

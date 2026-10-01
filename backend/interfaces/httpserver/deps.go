package httpserver

import (
	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/admin"
	"github.com/pauluno777/betailo/backend/application/usecase/auth"
	"github.com/pauluno777/betailo/backend/application/usecase/cycle"
	"github.com/pauluno777/betailo/backend/application/usecase/dashboard"
	"github.com/pauluno777/betailo/backend/application/usecase/farm"
	"github.com/pauluno777/betailo/backend/application/usecase/forum"
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
	Cookies      CookieConfig

	CreateFarm  *farm.CreateFarm
	GetFarm     *farm.GetFarm
	ListMyFarms *farm.ListMyFarms
	UpdateFarm  *farm.UpdateFarm
	DeleteFarm  *farm.DeleteFarm

	CreateCycle       *cycle.CreateCycle
	GetCycle          *cycle.GetCycle
	ListCyclesByFarm  *cycle.ListCyclesByFarm
	UpdateCycleStatus *cycle.UpdateCycleStatus
	DeleteCycle       *cycle.DeleteCycle

	CreateListing       *listing.CreateListing
	GetListing          *listing.GetListing
	UpdateListing       *listing.UpdateListing
	UnpublishListing    *listing.UnpublishListing
	PresignListingPhoto *listing.PresignListingPhoto
	SetListingPhotos    *listing.SetListingPhotos
	SearchListings      *listing.SearchListings
	Storage             ports.ObjectStorage

	ReserveCycle           *reservation.ReserveCycle
	PayReservation         *reservation.PayReservation
	MarkDelivered          *reservation.MarkDelivered
	ConfirmDelivery        *reservation.ConfirmDelivery
	OpenDispute            *reservation.OpenDispute
	ListMyReservations     *reservation.ListMyReservations
	ListReservationsByFarm *reservation.ListReservationsByFarm

	CreateReview    *review.CreateReview
	ListFarmReviews *review.ListByFarm

	CreateReport *report.CreateReport

	CreateForumPost *forum.CreatePost
	ListForumPosts  *forum.ListPosts
	GetForumPost    *forum.GetPost
	AddForumReply   *forum.AddReply
	MarkBestAnswer  *forum.MarkBestAnswer

	SellerDashboard *dashboard.SellerDashboard

	AdminOverview        *admin.Overview
	AdminListUsers       *admin.ListUsers
	AdminVerifyUser      *admin.VerifyUser
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

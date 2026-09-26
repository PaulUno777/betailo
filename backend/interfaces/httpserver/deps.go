package httpserver

import (
	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/auth"
	"github.com/pauluno777/betailo/backend/application/usecase/cycle"
	"github.com/pauluno777/betailo/backend/application/usecase/farm"
	"github.com/pauluno777/betailo/backend/application/usecase/listing"
)

// Dependencies is everything NewRouter needs to wire routes to
// usecases. A struct, not a growing parameter list, so adding the next
// business module's usecase (see the roadmap) doesn't change every
// existing call to NewRouter.
type Dependencies struct {
	Register *auth.Register
	Login    *auth.Login
	Signer   ports.TokenSigner

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

	CreateListing         *listing.CreateListing
	GetListing            *listing.GetListing
	ListPublishedListings *listing.ListPublishedListings
	UpdateListing         *listing.UpdateListing
	UnpublishListing      *listing.UnpublishListing
}

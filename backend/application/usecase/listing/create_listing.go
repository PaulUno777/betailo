// Package listing implements CRUD for a marketplace listing — Phase 2
// of the roadmap. Like application/usecase/farm and .../cycle, every
// usecase here depends only on application/ports.
package listing

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// CreateListing is the usecase: validate input, confirm the caller owns
// the cycle's farm, and persist a new listing — always attached to a
// cycle. "Une annonce sans cycle est impossible" (Phase 2's DoD): there
// is no path through this usecase that produces a Listing with a nil
// CycleID.
type CreateListing struct {
	Farms    ports.FarmRepository
	Cycles   ports.CycleStore
	Listings ports.ListingStore
}

// CreateListingInput is what a caller (the HTTP handler) supplies.
// OwnerID comes from the authenticated user's token.
type CreateListingInput struct {
	CycleID     bson.ObjectID
	OwnerID     bson.ObjectID
	Title       string
	Description string
	UnitPrice   float64
}

// Execute returns domain.ErrNotFound if the cycle or its farm doesn't
// exist, domain.ErrForbidden if OwnerID isn't the farm's owner, and
// domain.ErrInvalid for a missing title or a negative price.
func (uc *CreateListing) Execute(ctx context.Context, in CreateListingInput) (*domain.Listing, error) {
	c, err := uc.Cycles.FindByID(ctx, in.CycleID)
	if err != nil {
		return nil, err
	}

	f, err := uc.Farms.FindByID(ctx, c.FarmID)
	if err != nil {
		return nil, err
	}
	if f.OwnerID != in.OwnerID {
		return nil, fmt.Errorf("listing: not the cycle's farm owner: %w", domain.ErrForbidden)
	}

	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, fmt.Errorf("listing: title is required: %w", domain.ErrInvalid)
	}
	if in.UnitPrice < 0 {
		return nil, fmt.Errorf("listing: unitPrice must not be negative: %w", domain.ErrInvalid)
	}

	cycleID := in.CycleID
	l := &domain.Listing{
		CycleID:     &cycleID,
		Title:       title,
		Description: strings.TrimSpace(in.Description),
		UnitPrice:   in.UnitPrice,
		Status:      domain.ListingPublished,
	}

	if err := uc.Listings.Create(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

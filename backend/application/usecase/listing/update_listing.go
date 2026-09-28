package listing

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// ownerOf resolves who owns a listing by following cycleId -> farmId ->
// ownerId — a listing never stores an owner directly, since it isn't
// one (see domain/listing.go); this chain is what UpdateListing and
// UnpublishListing both need to authorize a write.
func ownerOf(ctx context.Context, cycles ports.CycleStore, farms ports.FarmRepository, l *domain.Listing) (bson.ObjectID, error) {
	if l.CycleID == nil {
		// Unreachable through CreateListing (see its doc comment), but
		// handled explicitly rather than panicking on a nil dereference
		// if a document were ever malformed.
		return bson.ObjectID{}, fmt.Errorf("listing: has no cycle: %w", domain.ErrInvalid)
	}
	c, err := cycles.FindByID(ctx, *l.CycleID)
	if err != nil {
		return bson.ObjectID{}, err
	}
	f, err := farms.FindByID(ctx, c.FarmID)
	if err != nil {
		return bson.ObjectID{}, err
	}
	return f.OwnerID, nil
}

// UpdateListing is the usecase: update a listing's editable fields,
// after confirming the caller owns its cycle's farm.
type UpdateListing struct {
	Farms    ports.FarmRepository
	Cycles   ports.CycleStore
	Listings ports.ListingStore
}

// UpdateListingInput is what a caller (the HTTP handler) supplies.
type UpdateListingInput struct {
	ListingID   bson.ObjectID
	OwnerID     bson.ObjectID
	Title       string
	Description string
	UnitPrice   float64
	Unit        string
}

// Execute returns domain.ErrForbidden if OwnerID doesn't own the
// listing's cycle's farm, and domain.ErrInvalid for a missing title or
// a negative price.
func (uc *UpdateListing) Execute(ctx context.Context, in UpdateListingInput) (*domain.Listing, error) {
	l, err := uc.Listings.FindByID(ctx, in.ListingID)
	if err != nil {
		return nil, err
	}

	actualOwner, err := ownerOf(ctx, uc.Cycles, uc.Farms, l)
	if err != nil {
		return nil, err
	}
	if actualOwner != in.OwnerID {
		return nil, fmt.Errorf("listing: not the owner: %w", domain.ErrForbidden)
	}

	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, fmt.Errorf("listing: title is required: %w", domain.ErrInvalid)
	}
	if in.UnitPrice < 0 {
		return nil, fmt.Errorf("listing: unitPrice must not be negative: %w", domain.ErrInvalid)
	}

	l.Title = title
	l.Description = strings.TrimSpace(in.Description)
	l.UnitPrice = in.UnitPrice
	l.Unit = strings.TrimSpace(in.Unit)

	if err := uc.Listings.Update(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

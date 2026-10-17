package listing

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

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

	// Breed, AgeDays and DeliveryAvailable are left unchanged when nil,
	// so older clients that only send the core fields keep working.
	Breed             *string
	AgeDays           *int
	DeliveryAvailable *bool
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
		return nil, domain.Coded("listing.title_required", fmt.Errorf("listing: title is required: %w", domain.ErrInvalid))
	}
	if in.UnitPrice < 0 {
		return nil, domain.Coded("listing.price_positive", fmt.Errorf("listing: unitPrice must not be negative: %w", domain.ErrInvalid))
	}
	if utf8.RuneCountInString(title) > maxTitleLen || utf8.RuneCountInString(strings.TrimSpace(in.Description)) > maxDescriptionLen {
		return nil, fmt.Errorf("listing: title or description too long: %w", domain.ErrInvalid)
	}
	if in.Breed != nil {
		breed := strings.TrimSpace(*in.Breed)
		if utf8.RuneCountInString(breed) > maxBreedLen {
			return nil, fmt.Errorf("listing: breed must be at most %d characters: %w", maxBreedLen, domain.ErrInvalid)
		}
		l.Breed = breed
	}
	if in.AgeDays != nil {
		if *in.AgeDays < 0 || *in.AgeDays > maxAgeDays {
			return nil, fmt.Errorf("listing: age is out of range: %w", domain.ErrInvalid)
		}
		l.AgeDays = *in.AgeDays
	}
	if in.DeliveryAvailable != nil {
		l.DeliveryAvailable = *in.DeliveryAvailable
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

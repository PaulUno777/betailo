package listing

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/catalog"
	"github.com/pauluno777/betailo/backend/domain"
)

const (
	maxTitleLen       = 120
	maxDescriptionLen = 2000
	maxBreedLen       = 60
	maxAgeWeeks       = 520
)

// QuickCreate is the sell wizard ("Vendre mes animaux"): one call
// creates the cycle the animals belong to and the listing selling them,
// in one transaction. The farmer picks a species and the platform knows
// the rest (cycle type, pricing unit). The listing starts as a draft:
// photos are attached next, then Publish puts it live.
type QuickCreate struct {
	Farms     ports.FarmRepository
	Cycles    ports.CycleStore
	Listings  ports.ListingStore
	Sequencer ports.Sequencer
	Tx        ports.TxManager
	Clock     ports.Clock
	// Catalog gives the species: its cycle type and unit.
	Catalog ports.CatalogSource
}

type QuickCreateInput struct {
	FarmID      bson.ObjectID
	OwnerID     bson.ObjectID
	Species     string // a catalog species key
	Breed       string
	AgeWeeks    int
	Quantity    int
	UnitPrice   float64
	Title       string
	Description string
	// AvailableFrom is nil for "disponibles maintenant", or the date the
	// animals become available (the cycle is then planned until it).
	AvailableFrom     *time.Time
	DeliveryAvailable bool
}

type QuickCreateOutput struct {
	Cycle   *domain.Cycle
	Listing *domain.Listing
}

func (uc *QuickCreate) Execute(ctx context.Context, in QuickCreateInput) (*QuickCreateOutput, error) {
	cat, err := catalog.Of(ctx, uc.Catalog)
	if err != nil {
		return nil, err
	}
	sp, err := cat.SpeciesByKey(in.Species)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(in.Title)
	description := strings.TrimSpace(in.Description)
	breed := strings.TrimSpace(in.Breed)
	switch {
	case title == "":
		return nil, domain.Coded("listing.title_required", fmt.Errorf("listing: a title is required: %w", domain.ErrInvalid))
	case utf8.RuneCountInString(title) > maxTitleLen:
		return nil, fmt.Errorf("listing: title must be at most %d characters: %w", maxTitleLen, domain.ErrInvalid)
	case utf8.RuneCountInString(description) > maxDescriptionLen:
		return nil, fmt.Errorf("listing: description must be at most %d characters: %w", maxDescriptionLen, domain.ErrInvalid)
	case utf8.RuneCountInString(breed) > maxBreedLen:
		return nil, fmt.Errorf("listing: breed must be at most %d characters: %w", maxBreedLen, domain.ErrInvalid)
	case in.AgeWeeks < 0 || in.AgeWeeks > maxAgeWeeks:
		return nil, fmt.Errorf("listing: age is out of range: %w", domain.ErrInvalid)
	case in.Quantity <= 0:
		return nil, domain.Coded("listing.quantity_positive", fmt.Errorf("listing: quantity must be positive: %w", domain.ErrInvalid))
	case in.UnitPrice <= 0:
		return nil, domain.Coded("listing.price_positive", fmt.Errorf("listing: price must be positive: %w", domain.ErrInvalid))
	}

	f, err := uc.Farms.FindByID(ctx, in.FarmID)
	if err != nil {
		return nil, err
	}
	if f.OwnerID != in.OwnerID {
		return nil, fmt.Errorf("listing: not the farm owner: %w", domain.ErrForbidden)
	}

	now := uc.Clock.Now()
	status, startsAt := domain.CycleStatusInProgress, now
	if in.AvailableFrom != nil {
		day := now.Truncate(24 * time.Hour)
		if in.AvailableFrom.Before(day) {
			return nil, domain.Coded("listing.date_past", fmt.Errorf("listing: the availability date is in the past: %w", domain.ErrInvalid))
		}
		if in.AvailableFrom.After(now) {
			status, startsAt = domain.CycleStatusPlanned, *in.AvailableFrom
		}
	}

	seq, err := uc.Sequencer.Next(ctx, domain.CycleRefPrefix)
	if err != nil {
		return nil, err
	}
	c := &domain.Cycle{
		ID: bson.NewObjectID(), Reference: domain.FormatReference(domain.CycleRefPrefix, seq),
		FarmID: f.ID, Species: sp.Key, Type: sp.CycleType, Status: status,
		Capacity: in.Quantity, AvailableQuantity: in.Quantity, StartsAt: startsAt,
	}
	l := &domain.Listing{
		ID: bson.NewObjectID(), CycleID: &c.ID, Title: title, Description: description,
		UnitPrice: in.UnitPrice, Unit: sp.Unit.Key, Status: domain.ListingDraft,
		Breed: breed, AgeWeeks: in.AgeWeeks, DeliveryAvailable: in.DeliveryAvailable,
	}
	err = uc.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := uc.Cycles.Create(ctx, c); err != nil {
			return err
		}
		return uc.Listings.Create(ctx, l)
	})
	if err != nil {
		return nil, err
	}
	return &QuickCreateOutput{Cycle: c, Listing: l}, nil
}

// Publish puts a draft (or a previously unpublished) listing live. A
// listing needs at least one photo to go live, and its cycle must still
// be open: there's nothing to sell from a finished or cancelled cycle.
// Its farm must be approved (farm.not_approved otherwise); that is
// checked last, so a seller first hears about what they can fix
// themselves.
type Publish struct {
	Farms    ports.FarmRepository
	Cycles   ports.CycleStore
	Listings ports.ListingStore
}

func (uc *Publish) Execute(ctx context.Context, listingID, ownerID bson.ObjectID) (*domain.Listing, error) {
	l, err := uc.Listings.FindByID(ctx, listingID)
	if err != nil {
		return nil, err
	}
	owner, err := ownerOf(ctx, uc.Cycles, uc.Farms, l)
	if err != nil {
		return nil, err
	}
	if owner != ownerID {
		return nil, fmt.Errorf("listing: not the owner: %w", domain.ErrForbidden)
	}
	if l.Status == domain.ListingPublished {
		return l, nil
	}
	if len(l.Photos) < MinListingPhotos {
		return nil, domain.Coded("listing.photo_required", fmt.Errorf("listing: add at least %d photo before publishing: %w", MinListingPhotos, domain.ErrInvalid))
	}
	c, err := uc.Cycles.FindByID(ctx, *l.CycleID)
	if err != nil {
		return nil, err
	}
	if c.Status != domain.CycleStatusPlanned && c.Status != domain.CycleStatusInProgress {
		return nil, domain.Coded("listing.cycle_closed", fmt.Errorf("listing: its cycle is closed: %w", domain.ErrConflict))
	}
	f, err := uc.Farms.FindByID(ctx, c.FarmID)
	if err != nil {
		return nil, err
	}
	if !f.Status.CanSell() {
		return nil, errFarmNotApproved(f)
	}
	l.Status = domain.ListingPublished
	if err := uc.Listings.Update(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

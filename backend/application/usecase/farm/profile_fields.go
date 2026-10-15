package farm

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/catalog"
	"github.com/pauluno777/betailo/backend/domain"
)

// place is where a farm is, as it stores it: the catalog's keys for a
// listed region and town, the typed text otherwise.
type place struct {
	Country domain.CatalogCountry
	Region  string
	Town    string
}

// normalizePlace validates where the farm is against the catalog: one
// of its active countries (the default when empty), a region that, for
// a country with a region list, must be one of its regions, and a town
// stored as its key when the catalog lists it (which also settles the
// region), as typed otherwise.
func normalizePlace(ctx context.Context, src ports.CatalogSource, country, region, town string) (place, error) {
	cat, err := catalog.Of(ctx, src)
	if err != nil {
		return place{}, err
	}
	c, err := cat.Country(country)
	if err != nil {
		return place{}, err
	}
	r, err := c.CanonicalRegion(region)
	if err != nil {
		return place{}, err
	}
	t, r, err := c.CanonicalCity(r, town)
	if err != nil {
		return place{}, err
	}
	return place{Country: c, Region: r, Town: t}, nil
}

// normalizeProfileFields validates the optional public-profile fields
// shared by create and update: a bounded description and a contact
// phone stored in E.164, read as a national number of the farm's
// country (callingCode) when typed without its "+".
func normalizeProfileFields(description, phone, callingCode string) (string, string, error) {
	description = strings.TrimSpace(description)
	if utf8.RuneCountInString(description) > domain.MaxFarmDescriptionLen {
		return "", "", fmt.Errorf("farm: description must be at most %d characters: %w", domain.MaxFarmDescriptionLen, domain.ErrInvalid)
	}
	if strings.TrimSpace(phone) == "" {
		return description, "", nil
	}
	normalized, err := domain.NormalizePhoneIn(phone, callingCode)
	if err != nil {
		return "", "", fmt.Errorf("farm: contact phone: %w", err)
	}
	return description, normalized, nil
}

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

// normalizeSpecies checks what the farm raises: at least one species,
// each an active one of the catalog, kept once, in the given order.
func normalizeSpecies(ctx context.Context, src ports.CatalogSource, species []string) ([]string, error) {
	cat, err := catalog.Of(ctx, src)
	if err != nil {
		return nil, err
	}
	var out []string
	seen := map[string]bool{}
	for _, key := range species {
		if seen[key] {
			continue
		}
		sp, err := cat.SpeciesByKey(key)
		if err != nil {
			return nil, err
		}
		seen[key] = true
		out = append(out, sp.Key)
	}
	if len(out) == 0 {
		return nil, domain.Coded("farm.species_required", fmt.Errorf("farm: pick at least one species: %w", domain.ErrInvalid))
	}
	return out, nil
}

// normalizeProfileFields validates the optional public-profile fields
// shared by create and update: a bounded description and up to max
// contact numbers, each stored in E.164 and read as a national number
// of the farm's country (callingCode) when typed without its "+".
// Blank numbers are dropped; the same number twice is refused.
func normalizeProfileFields(description string, contacts []domain.FarmContact, callingCode string, max int) (string, []domain.FarmContact, error) {
	description = strings.TrimSpace(description)
	if utf8.RuneCountInString(description) > domain.MaxFarmDescriptionLen {
		return "", nil, fmt.Errorf("farm: description must be at most %d characters: %w", domain.MaxFarmDescriptionLen, domain.ErrInvalid)
	}
	var out []domain.FarmContact
	seen := map[string]bool{}
	for _, c := range contacts {
		if strings.TrimSpace(c.Phone) == "" {
			continue
		}
		normalized, err := domain.NormalizePhoneIn(c.Phone, callingCode)
		if err != nil {
			return "", nil, fmt.Errorf("farm: contact phone: %w", err)
		}
		if seen[normalized] {
			return "", nil, domain.Coded("farm.contact_duplicate", fmt.Errorf("farm: contact %s is listed twice: %w", normalized, domain.ErrInvalid))
		}
		seen[normalized] = true
		out = append(out, domain.FarmContact{Phone: normalized, WhatsApp: c.WhatsApp})
	}
	if len(out) > max {
		return "", nil, domain.CodedWith("farm.contact_limit", map[string]any{"max": max}, fmt.Errorf("farm: at most %d contact numbers: %w", max, domain.ErrInvalid))
	}
	return description, out, nil
}

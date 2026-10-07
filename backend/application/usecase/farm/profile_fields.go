package farm

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/pauluno777/betailo/backend/domain"
)

// normalizePlace validates where the farm is: a country code (the
// default when empty) and a region that, for a country with a known
// list, must be one of its regions.
func normalizePlace(country, region string) (string, string, error) {
	c, err := domain.NormalizeCountry(country)
	if err != nil {
		return "", "", err
	}
	r, err := domain.CanonicalRegion(c, region)
	if err != nil {
		return "", "", err
	}
	return c, r, nil
}

// normalizeProfileFields validates the optional public-profile fields
// shared by create and update: a bounded description and a contact
// phone stored in E.164, read as a national number of the farm's
// country when typed without its "+".
func normalizeProfileFields(description, phone, country string) (string, string, error) {
	description = strings.TrimSpace(description)
	if utf8.RuneCountInString(description) > domain.MaxFarmDescriptionLen {
		return "", "", fmt.Errorf("farm: description must be at most %d characters: %w", domain.MaxFarmDescriptionLen, domain.ErrInvalid)
	}
	if strings.TrimSpace(phone) == "" {
		return description, "", nil
	}
	normalized, err := domain.NormalizePhoneIn(phone, country)
	if err != nil {
		return "", "", fmt.Errorf("farm: contact phone: %w", err)
	}
	return description, normalized, nil
}

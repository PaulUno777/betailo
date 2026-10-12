package farm

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/pauluno777/betailo/backend/domain"
)

// normalizeProfileFields validates the optional public-profile fields
// shared by create and update: a bounded description and a contact
// phone stored in E.164.
func normalizeProfileFields(description, phone string) (string, string, error) {
	description = strings.TrimSpace(description)
	if utf8.RuneCountInString(description) > domain.MaxFarmDescriptionLen {
		return "", "", fmt.Errorf("farm: description must be at most %d characters: %w", domain.MaxFarmDescriptionLen, domain.ErrInvalid)
	}
	if strings.TrimSpace(phone) == "" {
		return description, "", nil
	}
	normalized, err := domain.NormalizePhone(phone)
	if err != nil {
		return "", "", fmt.Errorf("farm: contact phone: %w", err)
	}
	return description, normalized, nil
}

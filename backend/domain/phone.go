package domain

import (
	"fmt"
	"strings"
)

// cameroonCountryCode is the default country: a number typed without
// one ("6 90 12 34 56") is read as Cameroonian, the platform's market.
const cameroonCountryCode = "237"

// NormalizePhone turns a phone number as people actually type it
// ("6 90 12 34 56", "690-12-34-56", "+237 690123456", "00237690123456")
// into E.164 ("+237690123456"). A bare 9-digit number is read as
// Cameroonian and must be a mobile (6xx) or fixed (2xx) line. Any other
// country needs its explicit "+" or "00" prefix, followed by 8-15 digits
// in total. Returns ErrInvalid otherwise.
func NormalizePhone(raw string) (string, error) {
	var b strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '+' && b.Len() == 0: // leading, possibly after "("
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '.' || r == '(' || r == ')':
			// Formatting only.
		default:
			return "", fmt.Errorf("phone: unexpected character %q: %w", r, ErrInvalid)
		}
	}
	s := b.String()

	switch {
	case strings.HasPrefix(s, "+"):
		s = s[1:]
	case strings.HasPrefix(s, "00"):
		s = s[2:]
	case len(s) == 9:
		s = cameroonCountryCode + s
	default:
		return "", fmt.Errorf("phone: add the country code (e.g. +237): %w", ErrInvalid)
	}

	if strings.HasPrefix(s, cameroonCountryCode) {
		local := s[len(cameroonCountryCode):]
		if len(local) != 9 || (local[0] != '6' && local[0] != '2') {
			return "", fmt.Errorf("phone: a Cameroonian number has 9 digits starting with 6 or 2: %w", ErrInvalid)
		}
	} else if len(s) < 8 || len(s) > 15 {
		return "", fmt.Errorf("phone: must have 8 to 15 digits: %w", ErrInvalid)
	}
	return "+" + s, nil
}

// MaskPhone hides the middle of an E.164 Cameroonian number for display
// ("+237690123456" -> "+237 6•• •• 34 56"). Other numbers keep only
// their last four digits.
func MaskPhone(e164 string) string {
	digits := strings.TrimPrefix(e164, "+")
	if strings.HasPrefix(digits, cameroonCountryCode) && len(digits) == len(cameroonCountryCode)+9 {
		local := digits[len(cameroonCountryCode):]
		return "+237 " + local[:1] + "•• •• " + local[5:7] + " " + local[7:]
	}
	if len(digits) <= 4 {
		return e164
	}
	return "•••• " + digits[len(digits)-4:]
}

// IsCameroonMobile reports whether an E.164 number is a Cameroonian
// mobile line, the only kind Mobile Money works with.
func IsCameroonMobile(e164 string) bool {
	return strings.HasPrefix(e164, "+"+cameroonCountryCode+"6") && len(e164) == 1+len(cameroonCountryCode)+9
}

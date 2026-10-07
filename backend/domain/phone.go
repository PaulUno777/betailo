package domain

import (
	"fmt"
	"strings"
)

// cameroonCountryCode is the default country: a number typed without
// one ("6 90 12 34 56") is read as Cameroonian, the platform's market.
const cameroonCountryCode = "237"

// CallingCodes maps the countries farms can be in (ISO 3166-1 alpha-2)
// to their international calling code, so a number typed the way people
// write it at home ("0244 123 456" in Ghana) can be read without its
// "+233".
var CallingCodes = map[string]string{
	"CM": "237", "BJ": "229", "BF": "226", "BI": "257", "CF": "236", "CG": "242", "CD": "243", "CI": "225",
	"DZ": "213", "EG": "20", "ET": "251", "GA": "241", "GH": "233", "GN": "224", "GQ": "240", "KE": "254",
	"MA": "212", "MG": "261", "ML": "223", "MR": "222", "MW": "265", "MZ": "258", "NE": "227", "NG": "234",
	"RW": "250", "SN": "221", "TD": "235", "TG": "228", "TN": "216", "TZ": "255", "UG": "256", "ZA": "27",
	"ZM": "260", "ZW": "263", "BE": "32", "CA": "1", "FR": "33", "US": "1",
}

// errPhone tags a phone refusal so clients can say what's wrong.
func errPhone(format string, args ...any) error {
	return Coded("phone.invalid", fmt.Errorf("phone: "+format+": %w", append(args, ErrInvalid)...))
}

// NormalizePhone is NormalizePhoneIn for the default country (Cameroon).
func NormalizePhone(raw string) (string, error) {
	return NormalizePhoneIn(raw, DefaultCountry)
}

// NormalizePhoneIn turns a phone number as people actually type it
// ("6 90 12 34 56", "690-12-34-56", "+237 690123456", "00237690123456")
// into E.164 ("+237690123456"). A number with its "+" or "00" prefix is
// read as given. Without one it's a national number of country: its
// trunk "0" is dropped and the country's calling code added. Cameroonian
// numbers must be 9 digits starting with 6 (mobile) or 2 (fixed); any
// other has 8-15 digits in total. Errors are ErrInvalid, coded
// "phone.invalid".
func NormalizePhoneIn(raw, country string) (string, error) {
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
			return "", errPhone("unexpected character %q", r)
		}
	}
	s := b.String()

	code, known := CallingCodes[country]
	switch {
	case strings.HasPrefix(s, "+"):
		s = s[1:]
	case strings.HasPrefix(s, "00"):
		s = s[2:]
	case !known:
		return "", errPhone("add the country code (e.g. +237)")
	case code == cameroonCountryCode && len(s) == 9:
		s = code + s
	case code != cameroonCountryCode && len(s) >= 6:
		s = code + strings.TrimPrefix(s, "0")
	default:
		return "", errPhone("add the country code (e.g. +%s)", code)
	}

	if strings.HasPrefix(s, cameroonCountryCode) {
		local := s[len(cameroonCountryCode):]
		if len(local) != 9 || (local[0] != '6' && local[0] != '2') {
			return "", errPhone("a Cameroonian number has 9 digits starting with 6 or 2")
		}
	} else if len(s) < 8 || len(s) > 15 {
		return "", errPhone("must have 8 to 15 digits")
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

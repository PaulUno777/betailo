package domain

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// DefaultCountry is the country a farm is in when none is given: the
// platform launched in Cameroon. Every other country works too.
const DefaultCountry = "CM"

// MaxRegionLen bounds a free-text region (countries without a list).
const MaxRegionLen = 60

// foldKey lowercases s and strips its accents, so "Extrême-Nord",
// "extreme-nord" and "EXTREME-NORD" compare equal.
func foldKey(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, err := transform.String(t, strings.ToLower(strings.TrimSpace(s)))
	if err != nil {
		return strings.ToLower(strings.TrimSpace(s))
	}
	return out
}

func isASCIIUpper(b byte) bool { return b >= 'A' && b <= 'Z' }

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

// MaxTownLen bounds a town typed freely (one the catalog doesn't list).
const MaxTownLen = 60

// foldKey lowercases s and strips its accents, so "Extrême-Nord",
// "far-north" and "FAR-NORTH" compare equal.
func foldKey(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	// Ligatures don't decompose: "œufs" and "oeufs" must still meet.
	s = strings.NewReplacer("œ", "oe", "æ", "ae").Replace(strings.ToLower(strings.TrimSpace(s)))
	out, _, err := transform.String(t, s)
	if err != nil {
		return s
	}
	return out
}

func isASCIIUpper(b byte) bool { return b >= 'A' && b <= 'Z' }

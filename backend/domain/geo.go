package domain

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// DefaultCountry is the country a farm is in when none is given: the
// platform launched in Cameroon. Every other country works too.
const DefaultCountry = "CM"

// MaxRegionLen bounds a free-text region (countries without a list).
const MaxRegionLen = 60

// KnownRegions lists the first-level regions of the countries Betailo
// knows in detail, in their usual written form. A farm in one of these
// countries must name one of its regions, so buyers can filter on an
// exact value. Any other country takes the region as typed: adding a
// country here only makes its filter stricter, it is never required.
var KnownRegions = map[string][]string{
	"CM": {
		"Adamaoua", "Centre", "Est", "Extrême-Nord", "Littoral",
		"Nord", "Nord-Ouest", "Ouest", "Sud", "Sud-Ouest",
	},
}

// NormalizeCountry returns an ISO 3166-1 alpha-2 code in upper case,
// DefaultCountry for an empty one, or ErrInvalid.
func NormalizeCountry(code string) (string, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return DefaultCountry, nil
	}
	if len(code) != 2 || !isASCIIUpper(code[0]) || !isASCIIUpper(code[1]) {
		return "", Coded("farm.country_invalid", fmt.Errorf("farm: country must be a 2-letter ISO code: %w", ErrInvalid))
	}
	return code, nil
}

// CanonicalRegion returns region in its canonical form for country: one
// of KnownRegions (matched ignoring case and accents) when the country
// has a list, else the trimmed text. Empty stays empty (a farm may not
// say yet).
func CanonicalRegion(country, region string) (string, error) {
	region = strings.Join(strings.Fields(region), " ")
	if region == "" {
		return "", nil
	}
	known, ok := KnownRegions[country]
	if !ok {
		if utf8.RuneCountInString(region) > MaxRegionLen {
			return "", Coded("farm.region_invalid", fmt.Errorf("farm: region must be at most %d characters: %w", MaxRegionLen, ErrInvalid))
		}
		return region, nil
	}
	key := foldKey(region)
	for _, r := range known {
		if foldKey(r) == key {
			return r, nil
		}
	}
	return "", Coded("farm.region_invalid", fmt.Errorf("farm: %q is not a region of %s: %w", region, country, ErrInvalid))
}

// cameroonTownRegions places Cameroon's main towns in their region, for
// filling in the region of farms created before regions existed.
var cameroonTownRegions = map[string]string{
	"ngaoundere": "Adamaoua", "meiganga": "Adamaoua", "tibati": "Adamaoua", "banyo": "Adamaoua",
	"yaounde": "Centre", "mbalmayo": "Centre", "obala": "Centre", "bafia": "Centre", "eseka": "Centre", "nanga eboko": "Centre", "mfou": "Centre",
	"bertoua": "Est", "batouri": "Est", "abong-mbang": "Est", "yokadouma": "Est",
	"maroua": "Extrême-Nord", "kousseri": "Extrême-Nord", "mokolo": "Extrême-Nord", "yagoua": "Extrême-Nord", "mora": "Extrême-Nord",
	"douala": "Littoral", "edea": "Littoral", "nkongsamba": "Littoral", "loum": "Littoral", "manjo": "Littoral", "mbanga": "Littoral", "yabassi": "Littoral",
	"garoua": "Nord", "guider": "Nord", "poli": "Nord", "tchollire": "Nord",
	"bamenda": "Nord-Ouest", "kumbo": "Nord-Ouest", "wum": "Nord-Ouest", "ndop": "Nord-Ouest", "nkambe": "Nord-Ouest",
	"bafoussam": "Ouest", "dschang": "Ouest", "mbouda": "Ouest", "foumban": "Ouest", "bafang": "Ouest", "bangangte": "Ouest", "foumbot": "Ouest", "bandjoun": "Ouest", "baham": "Ouest",
	"ebolowa": "Sud", "kribi": "Sud", "sangmelima": "Sud", "ambam": "Sud",
	"buea": "Sud-Ouest", "limbe": "Sud-Ouest", "kumba": "Sud-Ouest", "tiko": "Sud-Ouest", "mamfe": "Sud-Ouest",
}

// RegionForTown guesses the region of a town in country, or "" when it
// isn't known. Only used to backfill farms; farmers pick their region.
func RegionForTown(country, town string) string {
	if country != "CM" {
		return ""
	}
	return cameroonTownRegions[foldKey(town)]
}

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

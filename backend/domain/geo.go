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

package domain

import (
	"fmt"
	"slices"
)

// SpeciesInfo is what the platform knows about each kind of animal it
// sells: the cycle type it's raised in and the unit it's priced per.
// The values are the same keys the marketplace filters by.
type SpeciesInfo struct {
	Key       string
	CycleType string
	Unit      string
}

// Species is the closed list the sell wizard offers.
var Species = []SpeciesInfo{
	{Key: "poussins", CycleType: "couvee", Unit: "poussin"},
	{Key: "jeunes-poulets", CycleType: "engraissement", Unit: "sujet"},
	{Key: "pondeuses", CycleType: "ponte", Unit: "pondeuse"},
	{Key: "reproducteurs", CycleType: "reproduction", Unit: "sujet"},
	{Key: "porcs", CycleType: "engraissement", Unit: "porcelet"},
	{Key: "chevres", CycleType: "elevage", Unit: "chèvre"},
}

// LookupSpecies returns the species with key, or ErrInvalid.
func LookupSpecies(key string) (SpeciesInfo, error) {
	i := slices.IndexFunc(Species, func(s SpeciesInfo) bool { return s.Key == key })
	if i < 0 {
		return SpeciesInfo{}, fmt.Errorf("unknown species %q: %w", key, ErrInvalid)
	}
	return Species[i], nil
}

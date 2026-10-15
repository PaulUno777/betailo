package listing

import (
	"context"
	"math"
	"slices"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/application/usecase/catalog"
)

const (
	// minPriceSamples: below this, a "usual price" would just be one
	// farmer's price, so no hint is given.
	minPriceSamples = 3
	maxPriceSamples = 500
)

// PriceHint is the sell wizard's "Prix habituel pour ces animaux dans
// l'Ouest : 10 000 -> 14 000 FCFA": the middle half of what comparable
// listings ask, nearby when there are enough of them, otherwise across
// the platform.
type PriceHint struct {
	Search  ports.ListingSearchRepository
	Catalog ports.CatalogSource
}

type PriceHintOutput struct {
	Count       int
	Low, Median float64
	High        float64
	// Regional is false when there weren't enough listings in the region
	// and the range covers the whole platform.
	Regional bool
}

// Execute returns nil (and no error) when there aren't enough
// comparable listings for a meaningful range.
func (uc *PriceHint) Execute(ctx context.Context, species, region string) (*PriceHintOutput, error) {
	cat, err := catalog.Of(ctx, uc.Catalog)
	if err != nil {
		return nil, err
	}
	if _, err := cat.SpeciesByKey(species); err != nil {
		return nil, err
	}
	regional := region != ""
	prices, err := uc.Search.PublishedPrices(ctx, species, region, maxPriceSamples)
	if err != nil {
		return nil, err
	}
	if len(prices) < minPriceSamples && regional {
		regional = false
		if prices, err = uc.Search.PublishedPrices(ctx, species, "", maxPriceSamples); err != nil {
			return nil, err
		}
	}
	if len(prices) < minPriceSamples {
		return nil, nil
	}
	slices.Sort(prices)
	return &PriceHintOutput{
		Count: len(prices), Low: quantile(prices, 0.25), Median: quantile(prices, 0.5), High: quantile(prices, 0.75),
		Regional: regional,
	}, nil
}

// quantile interpolates linearly within sorted, then rounds to a whole
// FCFA amount (FCFA has no minor unit).
func quantile(sorted []float64, q float64) float64 {
	pos := q * float64(len(sorted)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	v := sorted[lo] + (sorted[hi]-sorted[lo])*(pos-float64(lo))
	return math.Round(v)
}

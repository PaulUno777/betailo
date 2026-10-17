package mongodb

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

var _ ports.ListingSearchRepository = (*ListingRepository)(nil)

const (
	defaultSearchPageSize = 20
	maxSearchPageSize     = 100
)

// searchRow is what each stage of the aggregation below decodes into —
// deliberately separate from listingDoc: this is a joined read model,
// not the listings collection's own document shape.
type searchRow struct {
	Listing listingDoc `bson:",inline"`
	Cycle   struct {
		Species  string             `bson:"species"`
		Type     string             `bson:"type"`
		Status   domain.CycleStatus `bson:"status"`
		Capacity int                `bson:"capacity"`
		Avail    int                `bson:"availableQuantity"`
		StartsAt time.Time          `bson:"startsAt"`
		EndsAt   *time.Time         `bson:"endsAt,omitempty"`
	} `bson:"cycle"`
	Farm struct {
		ID       bson.ObjectID `bson:"_id"`
		Name     string        `bson:"name"`
		Location string        `bson:"location"`
		Country  string        `bson:"country"`
		Region   string        `bson:"region"`
		// Verification.AutoApproved: let through by the test switch,
		// not reviewed by staff.
		Verification struct {
			AutoApproved bool `bson:"autoApproved"`
		} `bson:"verification"`
	} `bson:"farm"`
	Owner struct {
		TrustScore float64 `bson:"trustScore"`
	} `bson:"owner"`
	ReviewCount   int     `bson:"reviewCount"`
	ReviewAverage float64 `bson:"reviewAverage"`
}

// Search implements ports.ListingSearchRepository via one aggregation:
// join each published, unflagged listing to its cycle, farm and farm
// owner, apply the filters that aren't facets (query, trusted, dates),
// then one $facet computes the page, the total and every facet's counts
// — each facet with all the other dimension filters but its own — in a
// single round trip.
func (r *ListingRepository) Search(ctx context.Context, c ports.ListingSearchCriteria) (ports.ListingSearchResult, error) {
	match := bson.M{"status": domain.ListingPublished, "flagged": bson.M{"$ne": true}}
	if len(c.IDs) > 0 {
		match["_id"] = bson.M{"$in": c.IDs}
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$lookup", Value: bson.M{"from": "cycles", "localField": "cycleId", "foreignField": "_id", "as": "cycle"}}},
		{{Key: "$unwind", Value: "$cycle"}},
		{{Key: "$lookup", Value: bson.M{"from": "farms", "localField": "cycle.farmId", "foreignField": "_id", "as": "farm"}}},
		{{Key: "$unwind", Value: "$farm"}},
		// Only an approved farm sells: suspending one takes its
		// listings off the marketplace without touching them.
		{{Key: "$match", Value: bson.M{"farm.status": domain.FarmApproved}}},
		{{Key: "$lookup", Value: bson.M{"from": "users", "localField": "farm.ownerId", "foreignField": "_id", "as": "owner"}}},
		{{Key: "$unwind", Value: bson.M{"path": "$owner", "preserveNullAndEmptyArrays": true}}},
	}
	if common := commonConditions(c); len(common) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.M{"$and": common}}})
	}

	dims := dimensionConditions(c)
	except := func(skip ...string) bson.D {
		var conds bson.A
		for _, name := range dimensionOrder {
			if !slices.Contains(skip, name) && dims[name] != nil {
				conds = append(conds, dims[name])
			}
		}
		if len(conds) == 0 {
			return bson.D{{Key: "$match", Value: bson.M{}}}
		}
		return bson.D{{Key: "$match", Value: bson.M{"$and": conds}}}
	}
	all := except()

	reviews := bson.A{
		bson.D{{Key: "$lookup", Value: bson.M{"from": "reviews", "localField": "farm._id", "foreignField": "farmId", "as": "reviews"}}},
		bson.D{{Key: "$addFields", Value: bson.M{
			"reviewCount":   bson.M{"$size": "$reviews"},
			"reviewAverage": bson.M{"$ifNull": bson.A{bson.M{"$avg": "$reviews.rating"}, 0}},
		}}},
		bson.D{{Key: "$project", Value: bson.M{"reviews": 0}}},
	}

	// _id is always the last sort key: a stable tie-break keeps paging
	// deterministic (no listing shown twice or skipped across pages),
	// and on its own it means "newest first", since ObjectIDs embed
	// their creation time.
	var sort bson.D
	switch c.Sort {
	case ports.ListingSortRecent:
		sort = bson.D{{Key: "_id", Value: -1}}
	case ports.ListingSortPriceAsc:
		sort = bson.D{{Key: "unitPrice", Value: 1}, {Key: "_id", Value: -1}}
	case ports.ListingSortPriceDesc:
		sort = bson.D{{Key: "unitPrice", Value: -1}, {Key: "_id", Value: -1}}
	case ports.ListingSortRating:
		sort = bson.D{{Key: "reviewAverage", Value: -1}, {Key: "reviewCount", Value: -1}, {Key: "_id", Value: -1}}
	default:
		sort = bson.D{{Key: "cycle.startsAt", Value: 1}, {Key: "_id", Value: -1}}
	}

	page, pageSize := c.Page, c.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultSearchPageSize
	}
	if pageSize > maxSearchPageSize {
		pageSize = maxSearchPageSize
	}

	// Reviews are joined only for the page, unless the sort needs them.
	data := bson.A{all}
	if c.Sort == ports.ListingSortRating {
		data = append(data, reviews...)
	}
	data = append(data,
		bson.D{{Key: "$sort", Value: sort}},
		bson.D{{Key: "$skip", Value: (page - 1) * pageSize}},
		bson.D{{Key: "$limit", Value: pageSize}},
	)
	if c.Sort != ports.ListingSortRating {
		data = append(data, reviews...)
	}

	countBy := func(key any, skip ...string) bson.A {
		return bson.A{except(skip...),
			bson.D{{Key: "$group", Value: bson.M{"_id": key, "count": bson.M{"$sum": 1}}}},
			bson.D{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}, {Key: "_id", Value: 1}}}},
		}
	}
	pipeline = append(pipeline, bson.D{{Key: "$facet", Value: bson.M{
		"data":       data,
		"totalCount": bson.A{all, bson.D{{Key: "$count", Value: "count"}}},
		"species":    countBy("$cycle.species", "species"),
		"types":      countBy(bson.M{"$toLower": "$cycle.type"}, "types"),
		"statuses":   countBy("$cycle.status", "statuses"),
		// A town settles its region: the region list ignores both.
		"regions": countBy(bson.M{"country": "$farm.country", "region": "$farm.region"}, "place", "city"),
		"cities":  countBy(bson.M{"country": "$farm.country", "region": "$farm.region", "city": "$farm.location"}, "city"),
		"price": bson.A{except("price"), bson.D{{Key: "$group", Value: bson.M{
			"_id": nil, "min": bson.M{"$min": "$unitPrice"}, "max": bson.M{"$max": "$unitPrice"},
		}}}},
	}}})

	cur, err := r.Coll.Aggregate(ctx, pipeline)
	if err != nil {
		return ports.ListingSearchResult{}, err
	}
	defer cur.Close(ctx)

	type count struct {
		Value string `bson:"_id"`
		Count int    `bson:"count"`
	}
	var facet struct {
		Data       []searchRow `bson:"data"`
		TotalCount []struct {
			Count int `bson:"count"`
		} `bson:"totalCount"`
		Species  []count `bson:"species"`
		Types    []count `bson:"types"`
		Statuses []count `bson:"statuses"`
		Regions  []struct {
			ID struct {
				Country string `bson:"country"`
				Region  string `bson:"region"`
			} `bson:"_id"`
			Count int `bson:"count"`
		} `bson:"regions"`
		Cities []struct {
			ID struct {
				Country string `bson:"country"`
				Region  string `bson:"region"`
				City    string `bson:"city"`
			} `bson:"_id"`
			Count int `bson:"count"`
		} `bson:"cities"`
		Price []struct {
			Min float64 `bson:"min"`
			Max float64 `bson:"max"`
		} `bson:"price"`
	}
	if cur.Next(ctx) {
		if err := cur.Decode(&facet); err != nil {
			return ports.ListingSearchResult{}, err
		}
	}
	if err := cur.Err(); err != nil {
		return ports.ListingSearchResult{}, err
	}

	out := ports.ListingSearchResult{Items: make([]ports.ListingSearchItem, 0, len(facet.Data))}
	if len(facet.TotalCount) > 0 {
		out.Total = facet.TotalCount[0].Count
	}
	toCounts := func(in []count) []ports.FacetCount {
		res := make([]ports.FacetCount, 0, len(in))
		for _, c := range in {
			if c.Value != "" {
				res = append(res, ports.FacetCount{Value: c.Value, Count: c.Count})
			}
		}
		return res
	}
	out.Facets.Species = toCounts(facet.Species)
	out.Facets.CycleTypes = toCounts(facet.Types)
	out.Facets.Statuses = toCounts(facet.Statuses)
	out.Facets.Regions = make([]ports.RegionFacet, 0, len(facet.Regions))
	for _, rg := range facet.Regions {
		if rg.ID.Region != "" {
			out.Facets.Regions = append(out.Facets.Regions, ports.RegionFacet{Country: rg.ID.Country, Region: rg.ID.Region, Count: rg.Count})
		}
	}
	out.Facets.Cities = make([]ports.CityFacet, 0, len(facet.Cities))
	for _, ct := range facet.Cities {
		if ct.ID.City != "" {
			out.Facets.Cities = append(out.Facets.Cities, ports.CityFacet{Country: ct.ID.Country, Region: ct.ID.Region, City: ct.ID.City, Count: ct.Count})
		}
	}
	if len(facet.Price) > 0 {
		out.Facets.PriceMin, out.Facets.PriceMax = facet.Price[0].Min, facet.Price[0].Max
	}

	for _, row := range facet.Data {
		out.Items = append(out.Items, ports.ListingSearchItem{
			Listing:        row.Listing.toDomain(),
			Species:        row.Cycle.Species,
			CycleType:      row.Cycle.Type,
			CycleStatus:    row.Cycle.Status,
			StartsAt:       row.Cycle.StartsAt,
			EndsAt:         row.Cycle.EndsAt,
			Capacity:       row.Cycle.Capacity,
			Available:      row.Cycle.Avail,
			FarmID:         row.Farm.ID,
			FarmName:       row.Farm.Name,
			FarmLocation:   row.Farm.Location,
			FarmCountry:    row.Farm.Country,
			FarmRegion:     row.Farm.Region,
			FarmVerified:   !row.Farm.Verification.AutoApproved,
			FarmTrustScore: row.Owner.TrustScore,
			ReviewAverage:  row.ReviewAverage,
			ReviewCount:    row.ReviewCount,
		})
	}
	return out, nil
}

// dimensionOrder lists the facet dimensions; each is left out of its
// own facet's counts.
var dimensionOrder = []string{"species", "types", "statuses", "place", "city", "price"}

// dimensionConditions is one $match condition per facet dimension (nil
// when that filter is off).
func dimensionConditions(c ports.ListingSearchCriteria) map[string]any {
	dims := map[string]any{}
	if len(c.Species) > 0 {
		dims["species"] = bson.M{"cycle.species": bson.M{"$in": c.Species}}
	}
	if len(c.CycleTypes) > 0 {
		lower := make(bson.A, 0, len(c.CycleTypes))
		for _, t := range c.CycleTypes {
			lower = append(lower, strings.ToLower(t))
		}
		dims["types"] = bson.M{"$expr": bson.M{"$in": bson.A{bson.M{"$toLower": "$cycle.type"}, lower}}}
	}
	statuses := c.Statuses
	if len(statuses) == 0 && len(c.IDs) == 0 {
		statuses = []domain.CycleStatus{domain.CycleStatusPlanned, domain.CycleStatusInProgress}
	}
	if len(statuses) > 0 {
		dims["statuses"] = bson.M{"cycle.status": bson.M{"$in": statuses}}
	}
	place := bson.M{}
	if c.Country != "" {
		place["farm.country"] = c.Country
	}
	if c.Region != "" {
		place["farm.region"] = c.Region
	}
	if len(place) > 0 {
		dims["place"] = place
	}
	if c.City != "" {
		dims["city"] = bson.M{"farm.location": c.City}
	}
	if c.MinPrice > 0 || c.MaxPrice > 0 {
		price := bson.M{}
		if c.MinPrice > 0 {
			price["$gte"] = c.MinPrice
		}
		if c.MaxPrice > 0 {
			price["$lte"] = c.MaxPrice
		}
		dims["price"] = bson.M{"unitPrice": price}
	}
	return dims
}

// commonConditions are the filters with no facet: they narrow every
// count alike.
func commonConditions(c ports.ListingSearchCriteria) bson.A {
	var conds bson.A
	for i, word := range strings.Fields(c.Query) {
		re := bson.M{"$regex": regexp.QuoteMeta(word), "$options": "i"}
		or := bson.A{
			bson.M{"title": re}, bson.M{"farm.name": re}, bson.M{"farm.location": re},
			bson.M{"farm.region": re}, bson.M{"cycle.species": re},
		}
		if i < len(c.QueryPlaces) && len(c.QueryPlaces[i]) > 0 {
			keys := c.QueryPlaces[i]
			or = append(or, bson.M{"farm.location": bson.M{"$in": keys}}, bson.M{"farm.region": bson.M{"$in": keys}})
		}
		conds = append(conds, bson.M{"$or": or})
	}
	if c.TrustedOnly {
		conds = append(conds, bson.M{"owner.trustScore": bson.M{"$gte": domain.TrustedFarmerScore}})
	}
	if !c.AvailableTo.IsZero() {
		conds = append(conds, bson.M{"cycle.startsAt": bson.M{"$lte": c.AvailableTo}})
	}
	if !c.AvailableFrom.IsZero() {
		conds = append(conds, bson.M{"$or": bson.A{
			bson.M{"cycle.endsAt": bson.M{"$exists": false}},
			bson.M{"cycle.endsAt": nil},
			bson.M{"cycle.endsAt": bson.M{"$gte": c.AvailableFrom}},
		}})
	}
	return conds
}

// PublishedPrices implements ports.ListingSearchRepository.
func (r *ListingRepository) PublishedPrices(ctx context.Context, species, region string, limit int) ([]float64, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"status": domain.ListingPublished, "flagged": bson.M{"$ne": true}}}},
		{{Key: "$lookup", Value: bson.M{"from": "cycles", "localField": "cycleId", "foreignField": "_id", "as": "cycle"}}},
		{{Key: "$unwind", Value: "$cycle"}},
		{{Key: "$match", Value: bson.M{"cycle.species": species}}},
	}
	if region != "" {
		pipeline = append(pipeline,
			bson.D{{Key: "$lookup", Value: bson.M{"from": "farms", "localField": "cycle.farmId", "foreignField": "_id", "as": "farm"}}},
			bson.D{{Key: "$unwind", Value: "$farm"}},
			bson.D{{Key: "$match", Value: bson.M{"farm.location": bson.M{"$regex": regexp.QuoteMeta(region), "$options": "i"}}}},
		)
	}
	pipeline = append(pipeline,
		bson.D{{Key: "$sort", Value: bson.D{{Key: "_id", Value: -1}}}},
		bson.D{{Key: "$limit", Value: limit}},
		bson.D{{Key: "$project", Value: bson.M{"unitPrice": 1}}},
	)
	cur, err := r.Coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		UnitPrice float64 `bson:"unitPrice"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	prices := make([]float64, 0, len(rows))
	for _, row := range rows {
		prices = append(prices, row.UnitPrice)
	}
	return prices, nil
}

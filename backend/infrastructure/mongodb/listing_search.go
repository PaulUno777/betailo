package mongodb

import (
	"context"
	"regexp"
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
	} `bson:"farm"`
	Owner struct {
		IsVerified bool    `bson:"isVerified"`
		TrustScore float64 `bson:"trustScore"`
	} `bson:"owner"`
	ReviewCount   int     `bson:"reviewCount"`
	ReviewAverage float64 `bson:"reviewAverage"`
}

// Search implements ports.ListingSearchRepository via one aggregation:
// join each published, unflagged listing to its cycle, farm, farm
// owner and review summary, apply every filter server-side, then page
// and sort — a single round trip rather than N+1 calls per listing
// card.
func (r *ListingRepository) Search(ctx context.Context, c ports.ListingSearchCriteria) (ports.ListingSearchResult, error) {
	match := bson.M{"status": domain.ListingPublished, "flagged": bson.M{"$ne": true}}
	if c.Query != "" {
		match["title"] = bson.M{"$regex": regexp.QuoteMeta(c.Query), "$options": "i"}
	}
	if c.MinPrice > 0 || c.MaxPrice > 0 {
		price := bson.M{}
		if c.MinPrice > 0 {
			price["$gte"] = c.MinPrice
		}
		if c.MaxPrice > 0 {
			price["$lte"] = c.MaxPrice
		}
		match["unitPrice"] = price
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$lookup", Value: bson.M{"from": "cycles", "localField": "cycleId", "foreignField": "_id", "as": "cycle"}}},
		{{Key: "$unwind", Value: "$cycle"}},
	}

	cycleMatch := bson.M{}
	if c.Species != "" {
		cycleMatch["cycle.species"] = c.Species
	}
	if c.CycleType != "" {
		cycleMatch["cycle.type"] = c.CycleType
	}
	if len(cycleMatch) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: cycleMatch}})
	}

	pipeline = append(pipeline,
		bson.D{{Key: "$lookup", Value: bson.M{"from": "farms", "localField": "cycle.farmId", "foreignField": "_id", "as": "farm"}}},
		bson.D{{Key: "$unwind", Value: "$farm"}},
	)
	if c.Region != "" {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.M{"farm.location": bson.M{"$regex": regexp.QuoteMeta(c.Region), "$options": "i"}}}})
	}

	pipeline = append(pipeline,
		bson.D{{Key: "$lookup", Value: bson.M{"from": "users", "localField": "farm.ownerId", "foreignField": "_id", "as": "owner"}}},
		bson.D{{Key: "$unwind", Value: bson.M{"path": "$owner", "preserveNullAndEmptyArrays": true}}},
	)
	if c.VerifiedOnly {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.M{"owner.isVerified": true}}})
	}

	pipeline = append(pipeline,
		bson.D{{Key: "$lookup", Value: bson.M{"from": "reviews", "localField": "farm._id", "foreignField": "farmId", "as": "reviews"}}},
		bson.D{{Key: "$addFields", Value: bson.M{
			"reviewCount":   bson.M{"$size": "$reviews"},
			"reviewAverage": bson.M{"$ifNull": bson.A{bson.M{"$avg": "$reviews.rating"}, 0}},
		}}},
	)

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

	pipeline = append(pipeline, bson.D{{Key: "$facet", Value: bson.M{
		"data": bson.A{
			bson.D{{Key: "$sort", Value: sort}},
			bson.D{{Key: "$skip", Value: (page - 1) * pageSize}},
			bson.D{{Key: "$limit", Value: pageSize}},
		},
		"totalCount": bson.A{bson.D{{Key: "$count", Value: "count"}}},
	}}})

	cur, err := r.Coll.Aggregate(ctx, pipeline)
	if err != nil {
		return ports.ListingSearchResult{}, err
	}
	defer cur.Close(ctx)

	var facet struct {
		Data       []searchRow `bson:"data"`
		TotalCount []struct {
			Count int `bson:"count"`
		} `bson:"totalCount"`
	}
	if cur.Next(ctx) {
		if err := cur.Decode(&facet); err != nil {
			return ports.ListingSearchResult{}, err
		}
	}
	if err := cur.Err(); err != nil {
		return ports.ListingSearchResult{}, err
	}

	total := 0
	if len(facet.TotalCount) > 0 {
		total = facet.TotalCount[0].Count
	}

	items := make([]ports.ListingSearchItem, 0, len(facet.Data))
	for _, row := range facet.Data {
		items = append(items, ports.ListingSearchItem{
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
			FarmVerified:   row.Owner.IsVerified,
			FarmTrustScore: row.Owner.TrustScore,
			ReviewAverage:  row.ReviewAverage,
			ReviewCount:    row.ReviewCount,
		})
	}

	return ports.ListingSearchResult{Items: items, Total: total}, nil
}

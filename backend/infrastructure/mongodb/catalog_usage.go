package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/pauluno777/betailo/backend/application/ports"
)

// CatalogUsage implements ports.CatalogUsage: how many cycles, farms,
// listings, posts, reviews and reports use each reference key, counted
// with one aggregation per collection.
type CatalogUsage struct {
	DB *mongo.Database
}

var _ ports.CatalogUsage = (*CatalogUsage)(nil)

// Usage implements ports.CatalogUsage.
func (u *CatalogUsage) Usage(ctx context.Context) (map[string]int, error) {
	out := map[string]int{}
	count := func(coll string, prefix func(bson.M) string, pipeline ...bson.D) error {
		cur, err := u.DB.Collection(coll).Aggregate(ctx, append(mongo.Pipeline{}, pipeline...))
		if err != nil {
			return err
		}
		var rows []struct {
			ID    bson.M `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cur.All(ctx, &rows); err != nil {
			return err
		}
		for _, row := range rows {
			if key := prefix(row.ID); key != "" {
				out[key] += row.Count
			}
		}
		return nil
	}
	str := func(m bson.M, k string) string { s, _ := m[k].(string); return s }
	group := func(id bson.M) bson.D {
		return bson.D{{Key: "$group", Value: bson.M{"_id": id, "count": bson.M{"$sum": 1}}}}
	}
	unwind := func(path string) bson.D { return bson.D{{Key: "$unwind", Value: path}} }
	steps := []error{
		count("cycles", func(m bson.M) string { return "species:" + str(m, "v") }, group(bson.M{"v": "$species"})),
		count("listings", func(m bson.M) string {
			if str(m, "v") == "" {
				return ""
			}
			return "unit:" + str(m, "v")
		}, group(bson.M{"v": "$unit"})),
		count("farms", func(m bson.M) string { return "country:" + str(m, "c") }, group(bson.M{"c": "$country"})),
		count("farms", func(m bson.M) string { return "region:" + str(m, "c") + ":" + str(m, "r") }, group(bson.M{"c": "$country", "r": "$region"})),
		count("farms", func(m bson.M) string { return "city:" + str(m, "c") + ":" + str(m, "l") }, group(bson.M{"c": "$country", "l": "$location"})),
		count("farms", func(m bson.M) string { return "farmReviewChecks:" + str(m, "v") }, unwind("$verification.checks"), group(bson.M{"v": "$verification.checks"})),
		count("forum_posts", func(m bson.M) string { return "forumCategories:" + str(m, "v") }, group(bson.M{"v": "$category"})),
		count("reviews", func(m bson.M) string { return "reviewTags:" + str(m, "v") }, unwind("$tags"), group(bson.M{"v": "$tags"})),
		count("reports", func(m bson.M) string { return "reportReasons:" + str(m, "v") }, group(bson.M{"v": "$reason"})),
	}
	for _, err := range steps {
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/pauluno777/betailo/backend/domain"
)

// BackfillFarmPlaces gives farms created before countries and regions
// existed the default country, and a region when their town is one
// domain.RegionForTown knows. Farms in other towns keep an empty region
// until their farmer picks one. Idempotent: only farms without a
// country are touched.
func BackfillFarmPlaces(ctx context.Context, database *mongo.Database) (int, error) {
	farms := database.Collection("farms")
	cur, err := farms.Find(ctx, bson.M{"country": bson.M{"$exists": false}})
	if err != nil {
		return 0, err
	}
	var docs []struct {
		ID       bson.ObjectID `bson:"_id"`
		Location string        `bson:"location"`
	}
	if err := cur.All(ctx, &docs); err != nil {
		return 0, err
	}
	for _, d := range docs {
		set := bson.M{"country": domain.DefaultCountry}
		if r := domain.RegionForTown(domain.DefaultCountry, d.Location); r != "" {
			set["region"] = r
		}
		if _, err := farms.UpdateOne(ctx, bson.M{"_id": d.ID, "country": bson.M{"$exists": false}}, bson.M{"$set": set}); err != nil {
			return 0, err
		}
	}
	return len(docs), nil
}

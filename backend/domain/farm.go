package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Farm manages a farmer's operation: location, species raised. No
// repository lives for it yet — see domain/user.go's comment.
type Farm struct {
	ID        bson.ObjectID
	OwnerID   bson.ObjectID
	Name      string
	Location  string
	Species   []string
	CreatedAt time.Time
}

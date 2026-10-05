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

	// Description is the farmer's own pitch, shown on the public farm
	// profile. Optional.
	Description string
	// ContactPhone is E.164 (see NormalizePhone). Optional. It powers
	// the "Contacter l'éleveur" WhatsApp/call link, and the API only
	// shows it to signed-in users.
	ContactPhone string
}

// MaxFarmDescriptionLen bounds Farm.Description.
const MaxFarmDescriptionLen = 1000

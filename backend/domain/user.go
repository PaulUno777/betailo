package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Role is one of the three account roles the platform recognizes.
// Values match schemas/users.json's $jsonSchema enum exactly — do not
// change these string literals without updating that file too, or
// MongoDB will reject any user whose role isn't one of these three
// values.
type Role string

const (
	RoleFarmer Role = "farmer"
	RoleBuyer  Role = "buyer"
	RoleAdmin  Role = "admin"
)

// User manages accounts: registration, login, profile, role. No
// repository lives for it yet — this module's usecases aren't built
// (see the roadmap); add application/ports + infrastructure/mongo
// entries once they are, following the pattern reservation already
// uses.
type User struct {
	ID           bson.ObjectID
	Email        string
	PasswordHash string
	Role         Role
	IsVerified   bool
	TrustScore   float64
	CreatedAt    time.Time
}

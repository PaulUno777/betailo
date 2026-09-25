package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Role is one of the three account roles the platform recognizes.
type Role string

const (
	RoleEleveur  Role = "eleveur"
	RoleAcheteur Role = "acheteur"
	RoleAdmin    Role = "admin"
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

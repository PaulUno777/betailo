package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// StaffRole is what a member of the platform's team may do in the back
// office. Everyone else has none: there is one kind of account, which
// buys, and sells once it has an approved farm (see Farm.Status).
type StaffRole string

const (
	StaffNone       StaffRole = ""
	StaffSuperAdmin StaffRole = "super_admin"
)

// IsStaff reports whether the role opens the back office at all.
func (r StaffRole) IsStaff() bool { return r != StaffNone }

// User manages accounts: registration, login, profile, role. No
// repository lives for it yet — this module's usecases aren't built
// (see the roadmap); add application/ports + infrastructure/mongo
// entries once they are, following the pattern reservation already
// uses.
type User struct {
	ID           bson.ObjectID
	Email        string
	PasswordHash string
	// StaffRole is StaffNone for everyone but the platform's team.
	// Staff accounts don't sell: they can't own a farm.
	StaffRole StaffRole
	// FarmLimit is how many farms this account may own when staff
	// raised it for them; 0 means the platform default applies.
	FarmLimit int

	// DisplayName is the one public-facing identity field this MVP
	// collects — forum posts, reviews and reservation lists need
	// *something* attributable to show; full profile fields (phone,
	// address) are out of scope until a module actually needs them.
	DisplayName string
	// Phone (E.164) and City are the buyer's own contact details,
	// shown to a farmer they've reserved from. Optional.
	Phone string
	City  string

	// Suspended blocks a user from creating farms, cycles, listings,
	// reservations, reviews, reports or forum posts — set by
	// application/usecase/admin's account actions, never by the user
	// themselves.
	Suspended  bool
	TrustScore float64
	CreatedAt  time.Time
}

// Profile field limits.
const (
	MaxDisplayNameLen = 60
	MaxCityLen        = 80
)

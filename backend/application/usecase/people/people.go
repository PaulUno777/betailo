// Package people resolves the accounts behind content for display: the
// public name and profile photo shown next to a forum post or reply.
package people

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
	"github.com/pauluno777/betailo/backend/domain"
)

// Person is what the public sees of an account: never its email.
type Person struct {
	Name      string // domain.PublicName of the display name
	AvatarKey string
}

// Directory looks accounts up for display.
type Directory struct {
	Users ports.UserRepository
}

// Lookup resolves each distinct ID once. An account that can't be read
// is left out: the content is shown without a name rather than not at
// all.
func (d *Directory) Lookup(ctx context.Context, ids []bson.ObjectID) map[bson.ObjectID]Person {
	out := make(map[bson.ObjectID]Person, len(ids))
	for _, id := range ids {
		if _, done := out[id]; done || id.IsZero() {
			continue
		}
		u, err := d.Users.FindByID(ctx, id)
		if err != nil {
			continue
		}
		out[id] = Person{Name: domain.PublicName(u.DisplayName), AvatarKey: u.AvatarKey}
	}
	return out
}

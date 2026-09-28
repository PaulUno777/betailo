package ports

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// TrustRecomputer lets another module trigger a user's trust-score
// recomputation without importing application/usecase/trust directly —
// the same "depend on the other module's public interface, never its
// code" rule the architecture document states for cross-module calls
// (mirrored elsewhere by depending on a port like ports.FarmRepository
// rather than the farm package). Implemented by
// application/usecase/trust.Recompute.
type TrustRecomputer interface {
	Execute(ctx context.Context, userID bson.ObjectID) error
}

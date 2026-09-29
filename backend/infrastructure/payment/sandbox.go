// Package payment provides the one PaymentProvider implementation this
// codebase has: a sandbox that always succeeds. There is no real
// Mobile Money or card-processor account to integrate against for a
// portfolio project, so this package exists to make that limitation an
// explicit, swappable seam — exactly how infrastructure/objectstorage
// isolates Cloudflare R2/SeaweedFS behind ports.ObjectStorage — rather
// than a TODO buried inside a usecase. Swapping in MTN MoMo's or Orange
// Money's real API later means writing one new type here; no usecase
// changes.
package payment

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/pauluno777/betailo/backend/application/ports"
)

// Sandbox implements ports.PaymentProvider without ever contacting a
// real payment network: Capture always succeeds and returns a
// synthetic provider reference; Refund always succeeds. Good enough to
// demonstrate the escrow state machine end-to-end.
type Sandbox struct{}

var _ ports.PaymentProvider = Sandbox{}

// Capture implements ports.PaymentProvider.
func (Sandbox) Capture(_ context.Context, reference string, _ float64, method ports.PaymentMethod) (string, error) {
	return fmt.Sprintf("sandbox_%s_%s", method, bson.NewObjectID().Hex()), nil
}

// Refund implements ports.PaymentProvider.
func (Sandbox) Refund(_ context.Context, _ string, _ float64) error {
	return nil
}

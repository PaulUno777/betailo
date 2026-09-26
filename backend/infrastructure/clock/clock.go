// Package clock implements application/ports.Clock with the real system
// clock — its own package so the composition root can inject it like
// any other infrastructure dependency, and tests can substitute a fake
// instead.
package clock

import (
	"time"

	"github.com/pauluno777/betailo/backend/application/ports"
)

// System implements ports.Clock via time.Now, in UTC.
type System struct{}

var _ ports.Clock = System{}

func (System) Now() time.Time { return time.Now().UTC() }

package ports

import "context"

// Sequencer hands out human references like "RSV-2610-0042" (see
// domain.FormatReference): the prefix, the year and month, then a
// strictly increasing, gap-tolerant number within that month. It is
// atomic: concurrent callers never receive the same reference.
type Sequencer interface {
	NextReference(ctx context.Context, prefix string) (string, error)
}

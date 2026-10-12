package ports

import "context"

// Sequencer hands out gap-tolerant, strictly increasing numbers per key
// (one counter for reservations, one for cycles), used for human
// references like "RSV-0042". Next is atomic: concurrent callers never
// receive the same value.
type Sequencer interface {
	Next(ctx context.Context, key string) (int64, error)
}

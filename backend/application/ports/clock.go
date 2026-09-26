package ports

import "time"

// Clock abstracts the current time so usecases are deterministic to
// test — see infrastructure/clock for the real implementation and
// application/usecase/auth's tests for why it matters (asserting
// CreatedAt/ExpiresAt without a real clock racing the test).
type Clock interface {
	Now() time.Time
}

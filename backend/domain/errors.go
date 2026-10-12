package domain

import "errors"

// Sentinel domain errors — generic across entities, matched with
// errors.Is by callers (usecases, then the delivery layer, which maps
// them to HTTP status codes). Keeping them here, not per-package, means
// a usecase never needs to know which infrastructure produced a
// failure to react to it correctly.
var (
	ErrNotFound             = errors.New("not_found")
	ErrConflict             = errors.New("conflict")
	ErrInvalid              = errors.New("invalid")
	ErrInsufficientCapacity = errors.New("insufficient_capacity")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrEmailTaken           = errors.New("email_taken")
	ErrForbidden            = errors.New("forbidden")

	// ErrSuspended is an account an admin has suspended. Reported only
	// after the password checks out, so it reveals nothing to a guesser.
	ErrSuspended = errors.New("account suspended")
)

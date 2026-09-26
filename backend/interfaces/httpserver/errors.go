package httpserver

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/pauluno777/betailo/backend/domain"
)

// errUnauthorized is what RequireAuth reports for a missing, malformed,
// or invalid token — wrapping domain.ErrUnauthorized so writeError maps
// it the same way a usecase-level authorization failure would.
var errUnauthorized = fmt.Errorf("httpserver: unauthorized: %w", domain.ErrUnauthorized)

// errBadRequestBody is what a handler reports when it can't decode the
// request body as JSON — wrapping domain.ErrInvalid so writeError maps
// it to 400, same as a usecase-level validation failure.
var errBadRequestBody = fmt.Errorf("httpserver: request body must be valid JSON: %w", domain.ErrInvalid)

// errorResponse is every error body's shape — one consistent structure
// so a client never has to guess whether an endpoint reports failures
// differently from another.
type errorResponse struct {
	Error string `json:"error"`
}

// writeError maps a domain sentinel error to an HTTP status and writes
// a structured JSON body — never a bare 500 with a stack trace, and
// never the raw Go error string for anything writeErrorStatus doesn't
// recognize (that would leak internals to the client).
func writeError(w http.ResponseWriter, err error) {
	writeJSON(w, errorStatus(err), errorResponse{Error: errorMessage(err)})
}

// errorStatus maps a domain sentinel error to its HTTP status. Errors
// that don't match any sentinel are treated as internal — the handler
// that produced them is expected to have logged the real cause before
// calling writeError.
func errorStatus(err error) int {
	switch {
	case errors.Is(err, domain.ErrInvalid):
		return http.StatusBadRequest
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrEmailTaken), errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrInsufficientCapacity):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// errorMessage is deliberately generic for anything that isn't a
// recognized domain error — the client gets "internal error", not
// whatever a MongoDB driver or bcrypt happened to say.
func errorMessage(err error) string {
	switch {
	case errors.Is(err, domain.ErrInvalid):
		return err.Error()
	case errors.Is(err, domain.ErrUnauthorized):
		return "invalid credentials"
	case errors.Is(err, domain.ErrNotFound):
		return "not found"
	case errors.Is(err, domain.ErrEmailTaken):
		return "email already registered"
	case errors.Is(err, domain.ErrConflict):
		return "conflict"
	case errors.Is(err, domain.ErrInsufficientCapacity):
		return "insufficient capacity"
	default:
		return "internal error"
	}
}

package domain

import "errors"

// CodedError gives a refusal a stable, machine-readable code ("listing.
// photo_required") on top of its sentinel (which still decides the HTTP
// status) and its English developer message. Clients translate the code;
// they never show the message, which is written for logs, not people.
type CodedError struct {
	Code string
	// Params are values the translated message needs ("at least {min}").
	Params map[string]any
	Err    error
}

func (e *CodedError) Error() string { return e.Err.Error() }
func (e *CodedError) Unwrap() error { return e.Err }

// Coded tags err with code. The sentinel err wraps keeps deciding the
// status, so tagging never changes behavior, only what clients can say.
func Coded(code string, err error) error {
	return &CodedError{Code: code, Err: err}
}

// CodedWith is Coded plus the values the client's message needs.
func CodedWith(code string, params map[string]any, err error) error {
	return &CodedError{Code: code, Params: params, Err: err}
}

// CodeOf returns the code err was tagged with, or "" when none was.
func CodeOf(err error) (string, map[string]any) {
	var ce *CodedError
	if errors.As(err, &ce) {
		return ce.Code, ce.Params
	}
	return "", nil
}

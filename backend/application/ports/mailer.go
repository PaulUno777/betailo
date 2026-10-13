package ports

import "context"

// Mail is one plain-text message to one address.
type Mail struct {
	To      string
	Subject string
	Text    string
}

// Mailer sends e-mail. Sending is best effort wherever it is called: a
// message that could not go out is logged, never a reason to undo what
// triggered it.
type Mailer interface {
	Send(ctx context.Context, m Mail) error
}

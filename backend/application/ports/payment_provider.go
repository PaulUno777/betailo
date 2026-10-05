package ports

import "context"

// PaymentMethod is how a buyer pays — the three the design covers today.
type PaymentMethod string

const (
	PaymentMethodMTNMoMo     PaymentMethod = "mtn_momo"
	PaymentMethodOrangeMoney PaymentMethod = "orange_money"
	PaymentMethodCard        PaymentMethod = "card"
)

// CaptureRequest is one payment to take.
type CaptureRequest struct {
	Reference string // our reservation reference, echoed on the provider's side
	Amount    float64
	Method    PaymentMethod
	// PayerPhone is the E.164 Mobile Money number to charge (MoMo and
	// Orange Money push a confirmation prompt to it). Empty for card.
	PayerPhone string
}

// PaymentProvider abstracts capturing and refunding a payment behind a
// small interface, the same way ObjectStorage abstracts a specific
// object-storage SDK — see infrastructure/payment for the concrete
// (sandbox) implementation and why a real MoMo/Orange Money/card
// gateway isn't wired in.
type PaymentProvider interface {
	// Capture takes payment and returns a provider-side reference to
	// record against the reservation. Implementations return
	// domain.ErrConflict for anything the caller should treat as
	// "payment failed" — never a provider-specific error type.
	Capture(ctx context.Context, req CaptureRequest) (providerRef string, err error)

	// Refund reverses a previously captured payment.
	Refund(ctx context.Context, providerRef string, amount float64) error
}

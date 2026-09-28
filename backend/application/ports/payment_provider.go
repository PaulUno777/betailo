package ports

import "context"

// PaymentMethod is how a buyer pays — the three the design covers today.
type PaymentMethod string

const (
	PaymentMethodMTNMoMo     PaymentMethod = "mtn_momo"
	PaymentMethodOrangeMoney PaymentMethod = "orange_money"
	PaymentMethodCard        PaymentMethod = "card"
)

// PaymentProvider abstracts capturing and refunding a payment behind a
// small interface, the same way ObjectStorage abstracts a specific
// object-storage SDK — see infrastructure/payment for the concrete
// (sandbox) implementation and why a real MoMo/Orange Money/card
// gateway isn't wired in.
type PaymentProvider interface {
	// Capture takes payment for amount via method and returns a
	// provider-side reference to record against the reservation.
	// Implementations return domain.ErrConflict for anything the caller
	// should treat as "payment failed" — never a provider-specific
	// error type.
	Capture(ctx context.Context, reference string, amount float64, method PaymentMethod) (providerRef string, err error)

	// Refund reverses a previously captured payment.
	Refund(ctx context.Context, providerRef string, amount float64) error
}

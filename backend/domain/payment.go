package domain

import "time"

// PaymentStatus is a Reservation's embedded payment sub-document
// status. It's an explicit state machine rather than a boolean "paid"
// flag, on purpose — see the payment/escrow section of the "Concepts &
// Architecture" document for the full state machine and its two
// alternative terminal states. Values match schemas/reservations.json's
// embedded $jsonSchema enum exactly — do not change these string
// literals without updating that file too, or MongoDB will reject any
// reservation whose payment.status isn't one of these five values.
type PaymentStatus string

const (
	PaymentPending  PaymentStatus = "pending"  // reservation created, payment not yet captured
	PaymentHeld     PaymentStatus = "held"     // captured, held in escrow
	PaymentReleased PaymentStatus = "released" // released to the farmer after confirmation
	PaymentFailed   PaymentStatus = "failed"   // the payment itself never succeeded
	PaymentRefunded PaymentStatus = "refunded" // disputed or cancelled after being held
)

// PaymentInfo models the simplified escrow flow held inside each
// Reservation — there is no separate "payments" collection. It is
// always read and written together with its parent Reservation, never
// on its own.
type PaymentInfo struct {
	Status     PaymentStatus
	Amount     float64
	HeldAt     *time.Time
	ReleasedAt *time.Time

	// Method is how the buyer paid ("mtn_momo", "orange_money", "card").
	Method string
	// PayerPhoneMasked is the Mobile Money number, masked for display
	// ("+237 6•• •• 34 56"). The full number is only ever handed to the
	// payment provider at capture time, never stored.
	PayerPhoneMasked string
	// ProviderRef is the payment provider's reference for the capture,
	// needed to refund it later.
	ProviderRef string
}

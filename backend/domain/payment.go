package domain

import "time"

// PaymentStatus is a Reservation's embedded payment sub-document
// status. It's an explicit state machine rather than a boolean "paid"
// flag, on purpose — see the "Payment / Escrow" section of the
// "Concepts & Architecture" document for the full state machine and
// its two alternative terminal states.
type PaymentStatus string

const (
	PaymentPending  PaymentStatus = "en_attente" // reservation created, payment not yet captured
	PaymentHeld     PaymentStatus = "retenu"     // captured, held in escrow
	PaymentReleased PaymentStatus = "libere"     // released to the eleveur after confirmation
	PaymentFailed   PaymentStatus = "echoue"     // the payment itself never succeeded
	PaymentRefunded PaymentStatus = "rembourse"  // disputed or cancelled after being held
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
}

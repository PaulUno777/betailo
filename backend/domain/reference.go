package domain

import "fmt"

// Reference prefixes, one counter each.
const (
	ReservationRefPrefix = "RSV"
	CycleRefPrefix       = "CYC"
)

// FormatReference renders counter value n under prefix: "RSV-0042".
// At least four digits, more once the counter outgrows them.
func FormatReference(prefix string, n int64) string {
	return fmt.Sprintf("%s-%04d", prefix, n)
}

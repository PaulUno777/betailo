package domain

import (
	"fmt"
	"time"
)

// Reference prefixes, one counter per prefix and month.
const (
	ReservationRefPrefix = "RSV"
	CycleRefPrefix       = "CYC"
	FarmRefPrefix        = "FRM"
	ReportRefPrefix      = "RPT"
)

// referenceZone is the platform's clock for references: West Africa
// Time (UTC+1, no daylight saving), where it launched, so a reservation
// made just after midnight on the 1st belongs to the new month.
var referenceZone = time.FixedZone("WAT", 60*60)

// ReferencePeriod is the counter a reference made at `at` draws from:
// the prefix and the year and month, "RSV-2610" for October 2026.
func ReferencePeriod(prefix string, at time.Time) string {
	return prefix + "-" + at.In(referenceZone).Format("0601")
}

// FormatReference renders counter value n of a period: "RSV-2610-0042",
// the 42nd reservation of October 2026. Within a prefix, references
// sort in the order they were made (four digits hold 9,999 a month,
// more digits beyond that). References made before this format
// ("RSV-0042") stay as they were and sort before every new one.
func FormatReference(period string, n int64) string {
	return fmt.Sprintf("%s-%04d", period, n)
}

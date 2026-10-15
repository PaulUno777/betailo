package domain

import (
	"slices"
	"testing"
	"time"
)

func TestReferencePeriod_UsesTheMonthInWestAfricaTime(t *testing.T) {
	// 23:30 UTC on 31 October is already 1 November in Douala.
	at := time.Date(2026, 10, 31, 23, 30, 0, 0, time.UTC)
	if got := ReferencePeriod(ReservationRefPrefix, at); got != "RSV-2611" {
		t.Fatalf("ReferencePeriod = %q, want RSV-2611", got)
	}
}

func TestFormatReference_SortsInTheOrderReferencesWereMade(t *testing.T) {
	made := []string{
		"RSV-0042", // before the dated format
		FormatReference("RSV-2609", 7),
		FormatReference("RSV-2610", 2),
		FormatReference("RSV-2610", 10),
		FormatReference("RSV-2701", 1),
	}
	if made[3] != "RSV-2610-0010" {
		t.Fatalf("FormatReference = %q, want RSV-2610-0010", made[3])
	}
	sorted := slices.Clone(made)
	slices.Sort(sorted)
	if !slices.Equal(sorted, made) {
		t.Fatalf("sorted %v, want creation order %v", sorted, made)
	}
}

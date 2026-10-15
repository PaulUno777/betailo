package domain

import (
	"testing"
)

func TestRegionForTown(t *testing.T) {
	if got := RegionForTown("CM", "Yaoundé"); got != "Centre" {
		t.Errorf("Yaoundé = %q, want Centre", got)
	}
	if got := RegionForTown("CM", "Ville inconnue"); got != "" {
		t.Errorf("unknown town = %q, want empty", got)
	}
	if got := RegionForTown("GA", "Libreville"); got != "" {
		t.Errorf("other country = %q, want empty", got)
	}
}

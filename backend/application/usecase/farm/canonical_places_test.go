package farm

import (
	"context"
	"testing"

	"github.com/pauluno777/betailo/backend/domain"
)

func TestCanonicalizePlaces_GivesOldFarmsTheCatalogKeys(t *testing.T) {
	farms := newFakeFarms()
	old := &domain.Farm{Status: domain.FarmApproved, Country: "CM", Region: "Ouest", Location: "Bafoussam"}
	unlisted := &domain.Farm{Status: domain.FarmDraft, Country: "CM", Region: "Ouest", Location: "Bamendjou"}
	abroad := &domain.Farm{Status: domain.FarmPending, Country: "GA", Region: "Estuaire", Location: "Libreville"}
	for _, f := range []*domain.Farm{old, unlisted, abroad} {
		_ = farms.Create(context.Background(), f)
	}

	n, err := CanonicalizePlaces(context.Background(), farms, nil)
	if err != nil || n != 2 {
		t.Fatalf("CanonicalizePlaces = %d, %v; want the two Cameroonian farms rewritten", n, err)
	}
	if old.Region != "west" || old.Location != "bafoussam" {
		t.Errorf("listed town: %s/%s, want ouest/bafoussam", old.Region, old.Location)
	}
	if unlisted.Region != "west" || unlisted.Location != "Bamendjou" {
		t.Errorf("unlisted town: %s/%s, want ouest/Bamendjou", unlisted.Region, unlisted.Location)
	}
	if abroad.Region != "Estuaire" || abroad.Location != "Libreville" {
		t.Errorf("abroad: %s/%s, want it untouched", abroad.Region, abroad.Location)
	}

	if n, _ := CanonicalizePlaces(context.Background(), farms, nil); n != 0 {
		t.Errorf("second run rewrote %d farms, want 0", n)
	}
}

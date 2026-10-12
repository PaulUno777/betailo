package domain

import (
	"errors"
	"testing"
)

func TestNormalizeCountry(t *testing.T) {
	for in, want := range map[string]string{"": DefaultCountry, " cm ": "CM", "ga": "GA", "SN": "SN"} {
		if got, err := NormalizeCountry(in); err != nil || got != want {
			t.Errorf("NormalizeCountry(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"CMR", "1A", "c", "é1"} {
		if _, err := NormalizeCountry(bad); !errors.Is(err, ErrInvalid) {
			t.Errorf("NormalizeCountry(%q) error = %v, want ErrInvalid", bad, err)
		}
	}
}

func TestCanonicalRegion(t *testing.T) {
	cases := []struct{ country, in, want string }{
		{"CM", "extreme-nord", "Extrême-Nord"},
		{"CM", "  OUEST ", "Ouest"},
		{"CM", "", ""},
		{"GA", "Estuaire", "Estuaire"}, // a country without a list takes any region
		{"SN", "  Dakar   Plateau ", "Dakar Plateau"},
	}
	for _, c := range cases {
		if got, err := CanonicalRegion(c.country, c.in); err != nil || got != c.want {
			t.Errorf("CanonicalRegion(%q, %q) = %q, %v; want %q", c.country, c.in, got, err, c.want)
		}
	}
	if _, err := CanonicalRegion("CM", "Atlantique"); !errors.Is(err, ErrInvalid) {
		t.Errorf("unknown Cameroonian region: error = %v, want ErrInvalid", err)
	}
}

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

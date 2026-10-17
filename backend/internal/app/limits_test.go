package app

import (
	"testing"

	"github.com/pauluno777/betailo/backend/domain"
)

func TestMaxFarmContacts(t *testing.T) {
	for _, tt := range []struct {
		env  string
		want int
	}{
		{"", domain.DefaultMaxFarmContacts},
		{"abc", domain.DefaultMaxFarmContacts},
		{"0", domain.DefaultMaxFarmContacts},
		{"5", 5},
	} {
		t.Setenv("MAX_FARM_CONTACTS", tt.env)
		if got := maxFarmContacts(); got != tt.want {
			t.Errorf("MAX_FARM_CONTACTS=%q: got %d, want %d", tt.env, got, tt.want)
		}
	}
}

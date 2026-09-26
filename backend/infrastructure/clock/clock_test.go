package clock

import (
	"testing"
	"time"
)

func TestSystem_Now(t *testing.T) {
	before := time.Now().UTC()
	got := System{}.Now()
	after := time.Now().UTC()

	if got.Before(before) || got.After(after) {
		t.Errorf("Now() = %v, want between %v and %v", got, before, after)
	}
	if got.Location() != time.UTC {
		t.Errorf("Location() = %v, want time.UTC", got.Location())
	}
}

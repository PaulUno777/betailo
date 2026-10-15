package domain

import (
	"errors"
	"testing"
)

func TestReviewRules(t *testing.T) {
	if err := (ReviewCriteria{Health: 5, Timeliness: 0}).Validate(); err != nil {
		t.Errorf("valid criteria rejected: %v", err)
	}
	if err := (ReviewCriteria{Conformity: 6}).Validate(); !errors.Is(err, ErrInvalid) {
		t.Errorf("score 6 accepted: %v", err)
	}

	tags, err := NormalizeReviewTags([]string{"on_time", " on_time", "good_value"}, ActiveKeys(SeedCatalog().ReviewTags))
	if err != nil || len(tags) != 2 {
		t.Errorf("NormalizeReviewTags = %v, %v; want the two distinct known tags", tags, err)
	}
	if _, err := NormalizeReviewTags([]string{"free_beer"}, ActiveKeys(SeedCatalog().ReviewTags)); !errors.Is(err, ErrInvalid) {
		t.Errorf("unknown tag accepted: %v", err)
	}

	for in, want := range map[string]string{"Joël Tchoupo": "Joël T.", "Aïcha": "Aïcha", "  ": "", "Marie Claire Élong": "Marie É."} {
		if got := PublicName(in); got != want {
			t.Errorf("PublicName(%q) = %q, want %q", in, got, want)
		}
	}
}

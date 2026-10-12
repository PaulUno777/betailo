package domain

import (
	"errors"
	"testing"
)

func TestCheckOwnedKeys(t *testing.T) {
	prefix := "listings/6abb0f3dcd905c20e75b87c9/"
	ok := prefix + "6ac3c1e70c7615f679083e6d.jpg"
	if err := CheckOwnedKeys(prefix, []string{ok}); err != nil {
		t.Fatalf("own key rejected: %v", err)
	}
	for _, bad := range []string{
		"listings/000000000000000000000000/6ac3c1e70c7615f679083e6d.jpg", // another listing
		"reviews/6abb0f3dcd905c20e75b87c9/6ac3c1e70c7615f679083e6d.jpg",  // another resource
		prefix + "../../secrets.jpg",
		prefix + "6ac3c1e70c7615f679083e6d.png",
		prefix,
	} {
		if err := CheckOwnedKeys(prefix, []string{bad}); !errors.Is(err, ErrInvalid) {
			t.Errorf("CheckOwnedKeys(%q) = %v, want ErrInvalid", bad, err)
		}
	}
}

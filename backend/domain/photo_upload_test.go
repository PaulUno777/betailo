package domain

import (
	"errors"
	"testing"
)

func TestCheckPhotoSize(t *testing.T) {
	if err := CheckPhotoSize(MaxPhotoBytes, MaxPhotoBytes); err != nil {
		t.Fatalf("size at the cap refused: %v", err)
	}
	for _, size := range []int64{0, -1, MaxPhotoBytes + 1} {
		if err := CheckPhotoSize(size, MaxPhotoBytes); !errors.Is(err, ErrInvalid) {
			t.Errorf("CheckPhotoSize(%d) = %v, want ErrInvalid", size, err)
		}
	}
	if code, params := CodeOf(CheckPhotoSize(MaxThumbBytes+1, MaxThumbBytes)); code != "photo.too_large" || params["max"] != int64(120) {
		t.Errorf("over the cap: code %q params %v", code, params)
	}
}

func TestThumbKey(t *testing.T) {
	if got := ThumbKey("listings/a/6ac3c1e70c7615f679083e6d.jpg"); got != "listings/a/6ac3c1e70c7615f679083e6d.thumb.jpg" {
		t.Errorf("ThumbKey = %q", got)
	}
}

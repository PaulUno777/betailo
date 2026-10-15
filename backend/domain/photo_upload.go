package domain

import (
	"fmt"
	"strings"
)

// Every photo is resized and re-encoded as JPEG in the browser before
// upload. These caps are what the bucket accepts: a presigned upload URL
// is signed for the exact size the client declared, so a declared size
// over the cap is refused here and a larger body is refused by storage.
const (
	MaxPhotoBytes int64 = 800 << 10
	// MaxThumbBytes caps the small copy cards and grids show, so a
	// marketplace page doesn't download full photos.
	MaxThumbBytes int64 = 120 << 10
)

// CheckPhotoSize refuses an upload whose declared size is missing or
// over max.
func CheckPhotoSize(size, max int64) error {
	if size <= 0 {
		return Coded("photo.size_required", fmt.Errorf("photo: declared size must be positive: %w", ErrInvalid))
	}
	if size > max {
		return CodedWith("photo.too_large", map[string]any{"max": max >> 10}, fmt.Errorf("photo: %d bytes over the %d cap: %w", size, max, ErrInvalid))
	}
	return nil
}

// ThumbKey is where a photo's small copy lives, next to it: derived
// rather than stored, so a resource keeps one list of keys.
func ThumbKey(key string) string {
	return strings.TrimSuffix(key, ".jpg") + ".thumb.jpg"
}

package domain

import (
	"fmt"
	"regexp"
)

// objectKeyID matches the random part of every key the API mints: a
// 24-hex ObjectID plus the .jpg extension.
var objectKeyID = regexp.MustCompile(`^[0-9a-f]{24}\.jpg$`)

// CheckOwnedKeys verifies every key was minted under prefix (for
// example "listings/<id>/"), i.e. came from this resource's own presign
// endpoint. Without it a client could attach any object in the bucket,
// someone else's photos included, and get signed download URLs for it.
func CheckOwnedKeys(prefix string, keys []string) error {
	for _, k := range keys {
		if len(k) <= len(prefix) || k[:len(prefix)] != prefix || !objectKeyID.MatchString(k[len(prefix):]) {
			return fmt.Errorf("photo key %q wasn't issued for this upload: %w", k, ErrInvalid)
		}
	}
	return nil
}

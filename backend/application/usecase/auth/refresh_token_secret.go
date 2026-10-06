package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// refreshSecretBytes is 256 bits of entropy — plenty for a bearer
// secret that's also rotated on every use, so a brute-force window
// never stays open long.
const refreshSecretBytes = 32

// newRefreshSecret returns a fresh, URL-safe random secret — the raw
// value that goes in the cookie. Only its hash (hashRefreshToken) is
// ever persisted, so a database read alone can't reconstruct it.
func newRefreshSecret() (string, error) {
	buf := make([]byte, refreshSecretBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("auth: generate refresh secret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// hashRefreshToken is plain SHA-256, not bcrypt: unlike a user's
// password, a refresh secret is already 256 bits of uniform random
// entropy, never typed by a human and never reused across accounts, so
// none of the reasons to pay bcrypt's deliberate slowness (defending a
// low-entropy, human-chosen, reused secret against offline guessing)
// apply here — a fast hash is correct, not a shortcut.
func hashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

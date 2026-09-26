package ports

// PasswordHasher abstracts password hashing so usecases never depend on
// a specific algorithm (bcrypt today) directly.
type PasswordHasher interface {
	// Hash returns a hash safe to store, given a plaintext password.
	Hash(password string) (string, error)

	// Compare returns nil if password matches hash, an error otherwise.
	// Never distinguish "hash malformed" from "password wrong" in the
	// returned error's meaning to a caller — both mean "reject".
	Compare(hash, password string) error
}

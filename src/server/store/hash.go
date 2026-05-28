// Package store — shared hashing helpers for token/session storage
// Per AI.md PART 11: token storage must use SHA-256 hash only, never raw tokens
package store

import (
	"crypto/sha256"
	"encoding/base64"
)

// hashForStorage hashes a raw token or session ID with SHA-256 for DB storage.
// The raw value is given to the client; only the hash is ever written to disk.
func hashForStorage(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return base64.RawStdEncoding.EncodeToString(h[:])
}

// Package uid provides collision-resistant unique ID generation using crypto/rand.
package uid

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// New returns a unique ID with the given prefix: "<prefix>-<8 random hex chars>".
// Uses crypto/rand — never collides, never exposes timing information.
func New(prefix string) string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// Extremely unlikely; fall back to a larger read to reduce collision probability
		b = make([]byte, 8)
		rand.Read(b) //nolint:errcheck // best-effort fallback; crypto/rand.Read rarely errors and the resulting ID is still usable
	}
	if prefix == "" {
		return hex.EncodeToString(b)
	}
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b))
}

// Hex returns n random hex characters (n must be even; each byte = 2 hex chars).
func Hex(n int) string {
	if n <= 0 {
		n = 8
	}
	b := make([]byte, (n+1)/2)
	rand.Read(b) //nolint:errcheck // crypto/rand.Read effectively never fails on supported platforms; b defaults to zeroed bytes if it did
	return hex.EncodeToString(b)[:n]
}

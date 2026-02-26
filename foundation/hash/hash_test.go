//go:build unit
// +build unit

package hash_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vortex-fintech/go-lib/foundation/hash"
)

func TestHashStringsCanonical(t *testing.T) {
	h1 := hash.HashStringsCanonical("foo", "bar")
	h2 := hash.HashStringsCanonical("foo", "bar")
	assert.Equal(t, h1, h2, "canonical hashes must be deterministic")

	h3 := hash.HashStringsCanonical("bar", "foo")
	assert.NotEqual(t, h1, h3, "canonical hashes must differ when order changes")

	assert.Len(t, hash.HashStringsCanonical(), 64)
}

func TestHashStringsCanonical_NoDelimiterAmbiguity(t *testing.T) {
	left := hash.HashStringsCanonical("a\x1Fb", "c")
	right := hash.HashStringsCanonical("a", "b\x1Fc")
	assert.NotEqual(t, left, right)
}

func TestHashStringsCanonical_EmptySemantics(t *testing.T) {
	emptyTuple := hash.HashStringsCanonical()
	oneEmptyPart := hash.HashStringsCanonical("")
	assert.NotEqual(t, emptyTuple, oneEmptyPart)
}

func TestHMACStringsCanonical(t *testing.T) {
	keyA := []byte("k1")
	keyB := []byte("k2")

	h1 := hash.HMACStringsCanonical(keyA, "foo", "bar")
	h2 := hash.HMACStringsCanonical(keyA, "foo", "bar")
	assert.Equal(t, h1, h2, "HMAC must be deterministic for same key/input")

	h3 := hash.HMACStringsCanonical(keyB, "foo", "bar")
	assert.NotEqual(t, h1, h3, "different keys must produce different HMAC")

	h4 := hash.HMACStringsCanonical(keyA, "bar", "foo")
	assert.NotEqual(t, h1, h4, "different input order must change HMAC")

	assert.Len(t, h1, 64)
}

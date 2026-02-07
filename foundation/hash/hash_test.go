//go:build unit
// +build unit

package hash_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vortex-fintech/go-lib/foundation/hash"
)

func TestHashStringsWithSep(t *testing.T) {
	hash1 := hash.HashStringsWithSep("foo", "bar")
	hash2 := hash.HashStringsWithSep("foo", "bar")
	assert.Equal(t, hash1, hash2, "Hashes must be equal for same input")

	hash3 := hash.HashStringsWithSep("bar", "foo")
	assert.NotEqual(t, hash1, hash3, "Hashes must differ when order changes")

	hash4 := hash.HashStringsWithSep()
	assert.Len(t, hash4, 64, "Hash of empty input should be 64 hex characters")
}

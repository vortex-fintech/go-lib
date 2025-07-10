package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const separator = "\x1F"

func HashStringsWithSep(parts ...string) string {
	data := strings.Join(parts, separator)
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}

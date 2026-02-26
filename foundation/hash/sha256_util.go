package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"io"
)

// HashStringsCanonical returns SHA-256 over length-prefixed parts.
// It is unambiguous for tuple encoding and safe when raw parts can contain separators.
func HashStringsCanonical(parts ...string) string {
	h := sha256.New()
	writeCanonical(h, parts)
	return hex.EncodeToString(h.Sum(nil))
}

// HMACStringsCanonical returns keyed HMAC-SHA256 over length-prefixed parts.
// Use this variant for sensitive or low-entropy identifiers.
func HMACStringsCanonical(key []byte, parts ...string) string {
	h := hmac.New(sha256.New, key)
	writeCanonical(h, parts)
	return hex.EncodeToString(h.Sum(nil))
}

func writeCanonical(w io.Writer, parts []string) {
	var buf [binary.MaxVarintLen64]byte

	n := binary.PutUvarint(buf[:], uint64(len(parts)))
	_, _ = w.Write(buf[:n])

	for _, p := range parts {
		n = binary.PutUvarint(buf[:], uint64(len(p)))
		_, _ = w.Write(buf[:n])
		_, _ = io.WriteString(w, p)
	}
}

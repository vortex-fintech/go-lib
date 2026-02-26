package hmac

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

var (
	ErrEmptySecret = errors.New("secret cannot be empty")
	ErrEmptyMAC    = errors.New("expected MAC cannot be empty")
)

func Compute(code string, secret []byte) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(code))
	return hex.EncodeToString(h.Sum(nil))
}

func Verify(code string, secret []byte, expectedMAC string) (bool, error) {
	if len(secret) == 0 {
		return false, ErrEmptySecret
	}
	if expectedMAC == "" {
		return false, ErrEmptyMAC
	}

	expected, err := hex.DecodeString(expectedMAC)
	if err != nil {
		return false, err
	}

	h := hmac.New(sha256.New, secret)
	h.Write([]byte(code))
	return hmac.Equal(h.Sum(nil), expected), nil
}

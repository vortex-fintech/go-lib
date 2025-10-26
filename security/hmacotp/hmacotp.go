package hmacotp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func Compute(code string, secret []byte) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(code))
	return hex.EncodeToString(h.Sum(nil))
}

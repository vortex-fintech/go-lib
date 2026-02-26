package hmac_test

import (
	"encoding/hex"
	"testing"

	"github.com/vortex-fintech/go-lib/security/hmac"
)

func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func TestCompute_KnownVectors(t *testing.T) {
	// Векторы посчитаны HMAC-SHA256(code, key)
	// code="123456", key="secret"
	want := "4a83854cf6f0112b4295bddd535a9b3fbe54a3f90e853b59d42e4bed553c55a4"
	got := hmac.Compute("123456", []byte("secret"))
	if got != want {
		t.Fatalf("Compute mismatch: got %s want %s", got, want)
	}

	// code="000000", key="secret"
	want = "949a402dd58bcb68ae2fc9a35e457e55e5cb76d08fe534792f3c4f47decab5fe"
	got = hmac.Compute("000000", []byte("secret"))
	if got != want {
		t.Fatalf("Compute mismatch: got %s want %s", got, want)
	}

	// code="123456", key="other"
	want = "3582eeb3e90ea1d654b9f22b33c293d456261fef2c41871ab68912918fd94937"
	got = hmac.Compute("123456", []byte("other"))
	if got != want {
		t.Fatalf("Compute mismatch: got %s want %s", got, want)
	}
}

func TestCompute_DifferentKeyDifferentMac(t *testing.T) {
	a := hmac.Compute("123456", []byte("k1"))
	b := hmac.Compute("123456", []byte("k2"))
	if a == b {
		t.Fatal("expected different MAC for different keys")
	}
}

func TestCompute_DifferentCodeDifferentMac(t *testing.T) {
	a := hmac.Compute("123456", []byte("secret"))
	b := hmac.Compute("123457", []byte("secret"))
	if a == b {
		t.Fatal("expected different MAC for different messages")
	}
}

func TestCompute_StableDeterministic(t *testing.T) {
	key := []byte("secret")
	a := hmac.Compute("135790", key)
	b := hmac.Compute("135790", key)
	if a != b {
		t.Fatal("expected deterministic output for same input")
	}
}

func TestVerify_ValidMAC(t *testing.T) {
	key := []byte("secret")
	code := "123456"
	mac := hmac.Compute(code, key)

	ok, err := hmac.Verify(code, key, mac)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected valid MAC")
	}
}

func TestVerify_InvalidMAC(t *testing.T) {
	key := []byte("secret")
	code := "123456"
	fakeMAC := "0000000000000000000000000000000000000000000000000000000000000000"

	ok, err := hmac.Verify(code, key, fakeMAC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected invalid MAC")
	}
}

func TestVerify_EmptySecret(t *testing.T) {
	ok, err := hmac.Verify("123456", []byte{}, "abc")
	if ok {
		t.Fatal("expected false for empty secret")
	}
	if err != hmac.ErrEmptySecret {
		t.Fatalf("expected ErrEmptySecret, got %v", err)
	}
}

func TestVerify_EmptyMAC(t *testing.T) {
	ok, err := hmac.Verify("123456", []byte("secret"), "")
	if ok {
		t.Fatal("expected false for empty MAC")
	}
	if err != hmac.ErrEmptyMAC {
		t.Fatalf("expected ErrEmptyMAC, got %v", err)
	}
}

func TestVerify_InvalidHex(t *testing.T) {
	ok, err := hmac.Verify("123456", []byte("secret"), "not-hex!@#")
	if ok {
		t.Fatal("expected false for invalid hex")
	}
	if err == nil {
		t.Fatal("expected error for invalid hex")
	}
}

func TestVerify_WrongKey(t *testing.T) {
	code := "123456"
	mac := hmac.Compute(code, []byte("secret"))

	ok, err := hmac.Verify(code, []byte("wrong"), mac)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected invalid MAC for wrong key")
	}
}

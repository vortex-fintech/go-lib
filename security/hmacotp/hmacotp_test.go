package hmacotp_test

import (
	"encoding/hex"
	"testing"

	"github.com/vortex-fintech/go-lib/security/hmacotp"
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
	got := hmacotp.Compute("123456", []byte("secret"))
	if got != want {
		t.Fatalf("Compute mismatch: got %s want %s", got, want)
	}

	// code="000000", key="secret"
	want = "949a402dd58bcb68ae2fc9a35e457e55e5cb76d08fe534792f3c4f47decab5fe"
	got = hmacotp.Compute("000000", []byte("secret"))
	if got != want {
		t.Fatalf("Compute mismatch: got %s want %s", got, want)
	}

	// code="123456", key="other"
	want = "3582eeb3e90ea1d654b9f22b33c293d456261fef2c41871ab68912918fd94937"
	got = hmacotp.Compute("123456", []byte("other"))
	if got != want {
		t.Fatalf("Compute mismatch: got %s want %s", got, want)
	}
}

func TestCompute_DifferentKeyDifferentMac(t *testing.T) {
	a := hmacotp.Compute("123456", []byte("k1"))
	b := hmacotp.Compute("123456", []byte("k2"))
	if a == b {
		t.Fatal("expected different MAC for different keys")
	}
}

func TestCompute_DifferentCodeDifferentMac(t *testing.T) {
	a := hmacotp.Compute("123456", []byte("secret"))
	b := hmacotp.Compute("123457", []byte("secret"))
	if a == b {
		t.Fatal("expected different MAC for different messages")
	}
}

func TestCompute_StableDeterministic(t *testing.T) {
	key := []byte("secret")
	a := hmacotp.Compute("135790", key)
	b := hmacotp.Compute("135790", key)
	if a != b {
		t.Fatal("expected deterministic output for same input")
	}
}

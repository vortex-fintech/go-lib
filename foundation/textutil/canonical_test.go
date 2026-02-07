package textutil

import (
	"errors"
	"testing"
)

func TestCanonicalizeStrict(t *testing.T) {
	t.Parallel()

	t.Run("collapses spaces and trims", func(t *testing.T) {
		got, err := CanonicalizeStrict("  hello  \u00a0world  ", CanonicalPolicy{MaxRunes: 64})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "hello world" {
			t.Fatalf("unexpected value: %q", got)
		}
	})

	t.Run("allow empty", func(t *testing.T) {
		got, err := CanonicalizeStrict("   ", CanonicalPolicy{MaxRunes: 64, AllowEmpty: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Fatalf("expected empty string, got %q", got)
		}
	})

	t.Run("invalid max runes", func(t *testing.T) {
		_, err := CanonicalizeStrict("a", CanonicalPolicy{MaxRunes: 0})
		if !errors.Is(err, ErrInvalidText) {
			t.Fatalf("expected ErrInvalidText, got %v", err)
		}
	})

	t.Run("rejects newline", func(t *testing.T) {
		_, err := CanonicalizeStrict("hello\nworld", CanonicalPolicy{MaxRunes: 64})
		if !errors.Is(err, ErrInvalidText) {
			t.Fatalf("expected ErrInvalidText, got %v", err)
		}
	})

	t.Run("rejects invalid utf8", func(t *testing.T) {
		_, err := CanonicalizeStrict(string([]byte{0xff, 'a'}), CanonicalPolicy{MaxRunes: 64})
		if !errors.Is(err, ErrInvalidText) {
			t.Fatalf("expected ErrInvalidText, got %v", err)
		}
	})

	t.Run("rejects format chars by default", func(t *testing.T) {
		_, err := CanonicalizeStrict("a\u200Db", CanonicalPolicy{MaxRunes: 64})
		if !errors.Is(err, ErrInvalidText) {
			t.Fatalf("expected ErrInvalidText, got %v", err)
		}
	})

	t.Run("allows format chars when enabled", func(t *testing.T) {
		got, err := CanonicalizeStrict("a\u200Db", CanonicalPolicy{MaxRunes: 64, AllowFormatCF: true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "a\u200db" {
			t.Fatalf("unexpected value: %q", got)
		}
	})

	t.Run("rejects when rune limit exceeded", func(t *testing.T) {
		_, err := CanonicalizeStrict("abcd", CanonicalPolicy{MaxRunes: 3})
		if !errors.Is(err, ErrInvalidText) {
			t.Fatalf("expected ErrInvalidText, got %v", err)
		}
	})
}

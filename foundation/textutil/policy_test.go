package textutil

import (
	"errors"
	"regexp"
	"testing"
	"unicode"
)

func TestNormalizeText_ValidatesPolicy(t *testing.T) {
	_, err := NormalizeText("hello", TextPolicy{MaxRunes: 0})
	if !errors.Is(err, ErrInvalidPolicy) {
		t.Fatalf("expected ErrInvalidPolicy, got %v", err)
	}
}

func TestNormalizeText_AllowNewlines(t *testing.T) {
	policy := TextPolicy{
		MinRunes:      1,
		MaxRunes:      100,
		AllowEmpty:    false,
		AllowNewlines: true,
	}

	out, err := NormalizeText("line1\nline2\nline3", policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "line1\nline2\nline3" {
		t.Fatalf("unexpected output: %q", out)
	}

	// Without AllowNewlines should fail
	policyNoNewlines := TextPolicy{
		MinRunes:      1,
		MaxRunes:      100,
		AllowEmpty:    false,
		AllowNewlines: false,
	}
	_, err = NormalizeText("line1\nline2", policyNoNewlines)
	if !errors.Is(err, ErrInvalidText) {
		t.Fatalf("expected ErrInvalidText, got %v", err)
	}
}

func TestNormalizeText_NormalizeNFKC(t *testing.T) {
	policy := TextPolicy{
		MinRunes:      1,
		MaxRunes:      100,
		AllowEmpty:    false,
		NormalizeNFKC: true,
	}

	// Full-width characters should be normalized
	out, err := NormalizeText("Ｈｅｌｌｏ", policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "Hello" {
		t.Fatalf("expected normalized output, got %q", out)
	}

	// Compatibility characters
	out, err = NormalizeText("ﬁnancial", policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "financial" {
		t.Fatalf("expected ligature expanded, got %q", out)
	}
}

func TestNormalizeText_AllowedCharset_LettersAndDigits(t *testing.T) {
	policy := TextPolicy{
		MinRunes:   1,
		MaxRunes:   100,
		AllowEmpty: false,
		AllowedCharset: &AllowedCharset{
			AllowLetters: true,
			AllowDigits:  true,
			AllowSpace:   true,
		},
	}

	out, err := NormalizeText("Hello World 123", policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "Hello World 123" {
		t.Fatalf("unexpected output: %q", out)
	}

	// Should reject special characters
	_, err = NormalizeText("Hello-World", policy)
	if !errors.Is(err, ErrInvalidText) {
		t.Fatalf("expected ErrInvalidText for special chars, got %v", err)
	}
}

func TestNormalizeText_AllowedCharset_WithExtra(t *testing.T) {
	policy := TextPolicy{
		MinRunes:   1,
		MaxRunes:   100,
		AllowEmpty: false,
		AllowedCharset: &AllowedCharset{
			AllowLetters: true,
			AllowDigits:  true,
			ExtraAllowed: "._-@",
		},
	}

	out, err := NormalizeText("user_name-123@test", policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "user_name-123@test" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestNormalizeText_AllowedCharset_Scripts(t *testing.T) {
	policy := TextPolicy{
		MinRunes:   1,
		MaxRunes:   100,
		AllowEmpty: false,
		AllowedCharset: &AllowedCharset{
			AllowLetters:   true,
			AllowedScripts: []*unicode.RangeTable{unicode.Latin},
		},
	}

	out, err := NormalizeText("Hello", policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "Hello" {
		t.Fatalf("unexpected output: %q", out)
	}

	// Should reject Cyrillic
	_, err = NormalizeText("Привет", policy)
	if !errors.Is(err, ErrInvalidText) {
		t.Fatalf("expected ErrInvalidText for Cyrillic, got %v", err)
	}
}

func TestNormalizeText_AllowedCharset_DisallowMixedScripts(t *testing.T) {
	policy := TextPolicy{
		MinRunes:   1,
		MaxRunes:   100,
		AllowEmpty: false,
		AllowedCharset: &AllowedCharset{
			AllowLetters:         true,
			AllowedScripts:       []*unicode.RangeTable{unicode.Latin, unicode.Cyrillic},
			DisallowMixedScripts: true,
		},
	}

	// Pure Latin is OK
	_, err := NormalizeText("Hello", policy)
	if err != nil {
		t.Fatalf("unexpected error for pure Latin: %v", err)
	}

	// Pure Cyrillic is OK
	_, err = NormalizeText("Привет", policy)
	if err != nil {
		t.Fatalf("unexpected error for pure Cyrillic: %v", err)
	}

	// Mixed scripts should fail
	_, err = NormalizeText("Hello Привет", policy)
	if !errors.Is(err, ErrInvalidText) {
		t.Fatalf("expected ErrInvalidText for mixed scripts, got %v", err)
	}
}

func TestNormalizeText_EnforcesMinRunesAndBytes(t *testing.T) {
	_, err := NormalizeText("ab", TextPolicy{MinRunes: 3, MaxRunes: 8, AllowEmpty: false})
	if !errors.Is(err, ErrInvalidText) {
		t.Fatalf("expected ErrInvalidText for MinRunes, got %v", err)
	}

	_, err = NormalizeText("abcd", TextPolicy{MinRunes: 1, MaxRunes: 8, MaxBytes: 3, AllowEmpty: false})
	if !errors.Is(err, ErrInvalidText) {
		t.Fatalf("expected ErrInvalidText for MaxBytes, got %v", err)
	}
}

func TestNormalizeText_EnforcesPattern(t *testing.T) {
	policy := TextPolicy{
		MinRunes:   1,
		MaxRunes:   16,
		AllowEmpty: false,
		Pattern:    regexp.MustCompile(`^[a-z ]+$`),
	}

	out, err := NormalizeText("  hello   world  ", policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "hello world" {
		t.Fatalf("unexpected output: %q", out)
	}

	_, err = NormalizeText("hello-123", policy)
	if !errors.Is(err, ErrInvalidText) {
		t.Fatalf("expected ErrInvalidText for pattern mismatch, got %v", err)
	}
}

func TestValidatePoliciesWithLimits(t *testing.T) {
	t.Run("valid policy within limit", func(t *testing.T) {
		err := ValidatePoliciesWithLimits(PolicyWithLimit{
			Field: "name",
			Policy: TextPolicy{
				MinRunes:   1,
				MaxRunes:   64,
				MaxBytes:   256,
				AllowEmpty: false,
			},
			HardLimit: 64,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid hard limit", func(t *testing.T) {
		err := ValidatePoliciesWithLimits(PolicyWithLimit{
			Field: "name",
			Policy: TextPolicy{
				MinRunes:   1,
				MaxRunes:   64,
				AllowEmpty: false,
			},
			HardLimit: 0,
		})
		if !errors.Is(err, ErrInvalidPolicy) {
			t.Fatalf("expected ErrInvalidPolicy, got %v", err)
		}
	})

	t.Run("max runes exceeds hard limit", func(t *testing.T) {
		err := ValidatePoliciesWithLimits(PolicyWithLimit{
			Field: "name",
			Policy: TextPolicy{
				MinRunes:   1,
				MaxRunes:   65,
				AllowEmpty: false,
			},
			HardLimit: 64,
		})
		if !errors.Is(err, ErrInvalidPolicy) {
			t.Fatalf("expected ErrInvalidPolicy, got %v", err)
		}
	})

	t.Run("max bytes exceeds hard limit multiplier", func(t *testing.T) {
		err := ValidatePoliciesWithLimits(PolicyWithLimit{
			Field: "name",
			Policy: TextPolicy{
				MinRunes:   1,
				MaxRunes:   64,
				MaxBytes:   257,
				AllowEmpty: false,
			},
			HardLimit: 64,
		})
		if !errors.Is(err, ErrInvalidPolicy) {
			t.Fatalf("expected ErrInvalidPolicy, got %v", err)
		}
	})
}

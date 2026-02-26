package textutil

import (
	"errors"
	"regexp"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

var ErrInvalidPolicy = errors.New("invalid policy")

type TextPolicy struct {
	MinRunes int
	MaxRunes int
	MaxBytes int

	NormalizeNFKC bool

	AllowEmpty    bool
	AllowNewlines bool

	AllowedCharset *AllowedCharset
	Pattern        *regexp.Regexp
}

type AllowedCharset struct {
	AllowLetters bool
	AllowDigits  bool
	AllowSpace   bool

	ExtraAllowed string

	AllowedScripts       []*unicode.RangeTable
	DisallowMixedScripts bool
}

func (p TextPolicy) Validate() error {
	if p.MaxRunes <= 0 {
		return ErrInvalidPolicy
	}
	if p.MinRunes < 0 || p.MinRunes > p.MaxRunes {
		return ErrInvalidPolicy
	}
	if p.AllowEmpty && p.MinRunes != 0 {
		return ErrInvalidPolicy
	}
	if !p.AllowEmpty && p.MinRunes == 0 {
		return ErrInvalidPolicy
	}
	if p.MaxBytes < 0 {
		return ErrInvalidPolicy
	}
	return nil
}

type PolicyWithLimit struct {
	Field     string
	Policy    TextPolicy
	HardLimit int
}

func ValidatePoliciesWithLimits(items ...PolicyWithLimit) error {
	for _, item := range items {
		if item.HardLimit <= 0 {
			return ErrInvalidPolicy
		}
		if err := item.Policy.Validate(); err != nil {
			return err
		}
		if item.Policy.MaxRunes > item.HardLimit {
			return ErrInvalidPolicy
		}
		if item.Policy.MaxBytes > 0 && item.Policy.MaxBytes > item.HardLimit*4 {
			return ErrInvalidPolicy
		}
	}
	return nil
}

// NormalizeText validates and canonicalizes text according to the policy.
func NormalizeText(s string, p TextPolicy) (string, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}

	// Apply NFKC normalization first if requested
	if p.NormalizeNFKC {
		s = norm.NFKC.String(s)
	}

	out, err := CanonicalizeStrict(s, CanonicalPolicy{
		MaxRunes:      p.MaxRunes,
		AllowEmpty:    p.AllowEmpty,
		AllowFormatCF: false,
		AllowNewlines: p.AllowNewlines,
	})
	if err != nil {
		return "", err
	}

	runes := utf8.RuneCountInString(out)
	if runes < p.MinRunes {
		return "", ErrInvalidText
	}
	if p.MaxBytes > 0 && len(out) > p.MaxBytes {
		return "", ErrInvalidText
	}

	// Validate charset if specified
	if p.AllowedCharset != nil {
		if err := validateCharset(out, p.AllowedCharset); err != nil {
			return "", err
		}
	}

	if p.Pattern != nil && !p.Pattern.MatchString(out) {
		return "", ErrInvalidText
	}

	return out, nil
}

func validateCharset(s string, cs *AllowedCharset) error {
	for _, r := range s {
		if !isRuneAllowed(r, cs) {
			return ErrInvalidText
		}
	}

	// Check for mixed scripts if required
	if cs.DisallowMixedScripts && len(cs.AllowedScripts) > 0 {
		if err := checkMixedScripts(s, cs.AllowedScripts); err != nil {
			return err
		}
	}

	return nil
}

func isRuneAllowed(r rune, cs *AllowedCharset) bool {
	// Check space first (only space, not all whitespace)
	if r == ' ' && cs.AllowSpace {
		return true
	}

	// Check newline (for multiline text)
	if r == '\n' {
		return true
	}

	// Check letters
	if cs.AllowLetters && unicode.IsLetter(r) {
		// If specific scripts are allowed, check them
		if len(cs.AllowedScripts) > 0 {
			for _, script := range cs.AllowedScripts {
				if unicode.Is(script, r) {
					return true
				}
			}
			return false
		}
		return true
	}

	// Check digits
	if cs.AllowDigits && unicode.IsDigit(r) {
		return true
	}

	// Check extra allowed characters
	for _, extra := range cs.ExtraAllowed {
		if r == extra {
			return true
		}
	}

	return false
}

func checkMixedScripts(s string, allowedScripts []*unicode.RangeTable) error {
	var foundScript *unicode.RangeTable
	for _, r := range s {
		if !unicode.IsLetter(r) {
			continue
		}
		for _, script := range allowedScripts {
			if unicode.Is(script, r) {
				if foundScript == nil {
					foundScript = script
				} else if foundScript != script {
					return ErrInvalidText
				}
				break
			}
		}
	}
	return nil
}

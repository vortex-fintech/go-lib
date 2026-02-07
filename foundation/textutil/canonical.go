package textutil

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

var ErrInvalidText = errors.New("invalid text")

type CanonicalPolicy struct {
	MaxRunes      int
	AllowEmpty    bool
	AllowFormatCF bool
}

func CanonicalizeStrict(s string, p CanonicalPolicy) (string, error) {
	if p.MaxRunes <= 0 {
		return "", ErrInvalidText
	}

	const bytesPerRuneCap = 8
	if maxB := p.MaxRunes * bytesPerRuneCap; maxB > 0 && len(s) > maxB {
		return "", ErrInvalidText
	}

	s = strings.TrimSpace(s)
	if s == "" {
		if p.AllowEmpty {
			return "", nil
		}
		return "", ErrInvalidText
	}

	var b strings.Builder
	if capBytes := p.MaxRunes * 4; capBytes > 0 && capBytes < len(s) {
		b.Grow(capBytes)
	} else if len(s) < p.MaxRunes*4 {
		b.Grow(len(s))
	}

	outRunes := 0
	prevSpace := false

	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			return "", ErrInvalidText
		}
		i += size

		switch r {
		case '\n', '\r', '\u0085', '\u2028', '\u2029':
			return "", ErrInvalidText
		}
		if unicode.IsControl(r) {
			return "", ErrInvalidText
		}
		if !p.AllowFormatCF && unicode.In(r, unicode.Cf) {
			return "", ErrInvalidText
		}

		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteByte(' ')
				outRunes++
				if outRunes > p.MaxRunes {
					return "", ErrInvalidText
				}
				prevSpace = true
			}
			continue
		}

		prevSpace = false
		b.WriteRune(r)
		outRunes++
		if outRunes > p.MaxRunes {
			return "", ErrInvalidText
		}
	}

	return strings.TrimSpace(b.String()), nil
}

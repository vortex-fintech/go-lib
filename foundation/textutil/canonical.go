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
	AllowNewlines bool
}

func CanonicalizeStrict(s string, p CanonicalPolicy) (string, error) {
	if p.MaxRunes <= 0 {
		return "", ErrInvalidText
	}

	const maxUTF8BytesPerRune = 4
	q, r := len(s)/maxUTF8BytesPerRune, len(s)%maxUTF8BytesPerRune
	if q > p.MaxRunes || (q == p.MaxRunes && r > 0) {
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

		isNewline := r == '\n' || r == '\r' || r == '\u0085' || r == '\u2028' || r == '\u2029'
		if isNewline {
			if !p.AllowNewlines {
				return "", ErrInvalidText
			}
			b.WriteRune('\n')
			outRunes++
			if outRunes > p.MaxRunes {
				return "", ErrInvalidText
			}
			prevSpace = false
			continue
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

	out := strings.TrimSpace(b.String())
	if out == "" {
		if p.AllowEmpty {
			return "", nil
		}
		return "", ErrInvalidText
	}

	return out, nil
}

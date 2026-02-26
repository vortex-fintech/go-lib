package piiutil

import "testing"

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "standard", in: "user@example.com", want: "u**r@example.com"},
		{name: "two-char local", in: "ab@example.com", want: "a*@example.com"},
		{name: "three-char local", in: "abc@example.com", want: "a*c@example.com"},
		{name: "long local", in: "john.doe@example.com", want: "j******e@example.com"},
		{name: "single-char local", in: "u@example.com", want: "u@example.com"},
		{name: "trim spaces", in: "  user@example.com  ", want: "u**r@example.com"},
		{name: "invalid token", in: "weird", want: "w***d"},
		{name: "invalid token short", in: "ab", want: "a*"},
		{name: "invalid token single", in: "x", want: "x"},
		{name: "unicode local", in: "юзер@example.com", want: "ю**р@example.com"},
		{name: "unicode two-char", in: "юю@example.com", want: "ю*@example.com"},
		{name: "unicode single local", in: "ю@example.com", want: "ю@example.com"},
		{name: "empty", in: "   ", want: ""},
		{name: "at sign only", in: "@", want: "@"},
		{name: "at sign at start", in: "@example.com", want: "@**********m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaskEmail(tt.in); got != tt.want {
				t.Fatalf("MaskEmail(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestMaskPhone_Boundaries(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "one digit", in: "1", want: "1"},
		{name: "three digits", in: "123", want: "**3"},
		{name: "exactly four digits", in: "1234", want: "***4"},
		{name: "formatted four digits", in: "+1234", want: "+***4"},
		{name: "trim spaces", in: "  +1234  ", want: "+***4"},
		{name: "more than four digits", in: "+1234567890", want: "+******7890"},
		{name: "mixed letters and short digits", in: "AB12", want: "AB*2"},
		{name: "no digits with separators", in: "AB-CD", want: "**-*D"},
		{name: "only separators", in: "()-", want: "()-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaskPhone(tt.in); got != tt.want {
				t.Fatalf("MaskPhone(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestMaskIDLast4_Boundaries(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "no digits", in: "ABCD", want: "***D"},
		{name: "two digits", in: "12-AB", want: "*2-AB"},
		{name: "exactly four digits", in: "1234", want: "***4"},
		{name: "formatted four digits", in: "AB-1234-CD", want: "AB-***4-CD"},
		{name: "more than four digits", in: "123-45-6789", want: "***-**-6789"},
		{name: "letters with long digits", in: "S1234567D", want: "S***4567D"},
		{name: "only separators", in: "----", want: "----"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaskIDLast4(tt.in); got != tt.want {
				t.Fatalf("MaskIDLast4(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestMaskDigitsKeepLast4Or1(t *testing.T) {
	t.Run("no digits returns false", func(t *testing.T) {
		runes := []rune("AB-CD")
		if ok := maskDigitsKeepLast4Or1(runes); ok {
			t.Fatalf("expected false when no digits")
		}
		if got := string(runes); got != "AB-CD" {
			t.Fatalf("expected unchanged runes, got %q", got)
		}
	})

	t.Run("short digits keep one", func(t *testing.T) {
		runes := []rune("1234")
		if ok := maskDigitsKeepLast4Or1(runes); !ok {
			t.Fatalf("expected true when digits exist")
		}
		if got := string(runes); got != "***4" {
			t.Fatalf("expected ***4, got %q", got)
		}
	})

	t.Run("long digits keep four", func(t *testing.T) {
		runes := []rune("1234567")
		if ok := maskDigitsKeepLast4Or1(runes); !ok {
			t.Fatalf("expected true when digits exist")
		}
		if got := string(runes); got != "***4567" {
			t.Fatalf("expected ***4567, got %q", got)
		}
	})
}

func TestMaskLettersAndDigitsKeepLast(t *testing.T) {
	t.Run("empty runes", func(t *testing.T) {
		if got := maskLettersAndDigitsKeepLast([]rune{}, 1); got != "" {
			t.Fatalf("expected empty string, got %q", got)
		}
	})

	t.Run("keep less than one defaults to one", func(t *testing.T) {
		if got := maskLettersAndDigitsKeepLast([]rune("AB12"), 0); got != "***2" {
			t.Fatalf("expected ***2, got %q", got)
		}
	})

	t.Run("keep greater than total keeps all", func(t *testing.T) {
		if got := maskLettersAndDigitsKeepLast([]rune("AB12"), 10); got != "AB12" {
			t.Fatalf("expected AB12, got %q", got)
		}
	})

	t.Run("only separators unchanged", func(t *testing.T) {
		if got := maskLettersAndDigitsKeepLast([]rune("-()"), 1); got != "-()" {
			t.Fatalf("expected -(), got %q", got)
		}
	})
}

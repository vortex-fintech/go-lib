package geo

import "testing"

func TestNormalizeISO2(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{name: "trim and uppercase", in: "  us  ", want: "US", ok: true},
		{name: "already normalized", in: "DE", want: "DE", ok: true},
		{name: "mixed case", in: "gB", want: "GB", ok: true},
		{name: "trim newline and tab", in: "\nca\t", want: "CA", ok: true},
		{name: "contains digit", in: "A1", want: "", ok: false},
		{name: "too short", in: "U", want: "", ok: false},
		{name: "too long", in: "USA", want: "", ok: false},
		{name: "internal space", in: "U S", want: "", ok: false},
		{name: "unicode sharp s", in: "\u00DF", want: "", ok: false},
		{name: "non ascii letters", in: "\u00E9\u00E9", want: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := NormalizeISO2(tt.in)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("NormalizeISO2(%q) = (%q, %t), want (%q, %t)", tt.in, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestIsValidISO2(t *testing.T) {
	if !IsValidISO2("zz") {
		t.Fatal("IsValidISO2(zz) should be true for format-only validation")
	}
	if IsValidISO2("z1") {
		t.Fatal("IsValidISO2(z1) should be false")
	}
	if IsValidISO2("\u00DF") {
		t.Fatal("IsValidISO2(\\u00DF) should be false")
	}
}

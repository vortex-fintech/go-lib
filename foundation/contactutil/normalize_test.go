package contactutil

import "testing"

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "trim and lowercase", in: "  User@Example.COM  ", want: "user@example.com"},
		{name: "invalid shape is preserved", in: "  not-an-email  ", want: "not-an-email"},
		{name: "empty after trim", in: "   ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeEmail(tt.in); got != tt.want {
				t.Fatalf("NormalizeEmail(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeE164(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "trim only", in: "  +1234567890  ", want: "+1234567890"},
		{name: "internal formatting is preserved", in: " +65 1234-5678 ", want: "+65 1234-5678"},
		{name: "non phone token is preserved", in: "  abc  ", want: "abc"},
		{name: "empty after trim", in: "   ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeE164(tt.in); got != tt.want {
				t.Fatalf("NormalizeE164(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

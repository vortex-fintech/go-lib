package textutil

import "testing"

func TestFirstNonEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []string
		want  string
	}{
		{
			name:  "returns first non-empty",
			input: []string{"", "a", "b"},
			want:  "a",
		},
		{
			name:  "skips whitespace",
			input: []string{"   ", "\t", "  b  "},
			want:  "b",
		},
		{
			name:  "returns empty when none",
			input: []string{"", "   ", "\n"},
			want:  "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := FirstNonEmpty(tt.input...); got != tt.want {
				t.Fatalf("FirstNonEmpty() = %q, want %q", got, tt.want)
			}
		})
	}
}

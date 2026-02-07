package netutil

import (
	"testing"
	"time"
)

func TestSanitizeTimeout(t *testing.T) {
	tests := []struct {
		name     string
		d        time.Duration
		min      time.Duration
		fallback time.Duration
		want     time.Duration
	}{
		{
			name:     "negative duration uses fallback",
			d:        -1 * time.Second,
			min:      100 * time.Millisecond,
			fallback: 30 * time.Second,
			want:     30 * time.Second,
		},
		{
			name:     "zero duration returns zero",
			d:        0,
			min:      0,
			fallback: 10 * time.Second,
			want:     0,
		},
		{
			name:     "duration below min returns min",
			d:        50 * time.Millisecond,
			min:      100 * time.Millisecond,
			fallback: 5 * time.Second,
			want:     100 * time.Millisecond,
		},
		{
			name:     "duration above min returns same value",
			d:        2 * time.Second,
			min:      1 * time.Second,
			fallback: 10 * time.Second,
			want:     2 * time.Second,
		},
		{
			name:     "min is zero ignored",
			d:        500 * time.Millisecond,
			min:      0,
			fallback: 10 * time.Second,
			want:     500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeTimeout(tt.d, tt.min, tt.fallback)
			if got != tt.want {
				t.Errorf("SanitizeTimeout(%v, %v, %v) = %v, want %v",
					tt.d, tt.min, tt.fallback, got, tt.want)
			}
		})
	}
}

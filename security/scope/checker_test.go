package scope_test

import (
	"testing"

	"github.com/vortex-fintech/go-lib/security/scope"
)

func TestIndex(t *testing.T) {
	t.Parallel()

	scopes := []string{"wallet:read", "wallet:write", "payments:create"}
	idx := scope.Index(scopes)

	if len(idx) != 3 {
		t.Fatalf("expected 3 items, got %d", len(idx))
	}

	for _, s := range scopes {
		if _, ok := idx[s]; !ok {
			t.Fatalf("expected scope %s in index", s)
		}
	}
}

func TestIndex_Empty(t *testing.T) {
	t.Parallel()

	idx := scope.Index(nil)
	if idx == nil {
		t.Fatal("expected non-nil map")
	}
	if len(idx) != 0 {
		t.Fatalf("expected empty map, got %d", len(idx))
	}
}

func TestIndex_Dedup(t *testing.T) {
	t.Parallel()

	scopes := []string{"a", "b", "a", "c", "b"}
	idx := scope.Index(scopes)

	if len(idx) != 3 {
		t.Fatalf("expected 3 unique items, got %d", len(idx))
	}
}

func TestHasAll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		scopes []string
		need   []string
		want   bool
	}{
		{"all present", []string{"a", "b", "c"}, []string{"a", "b"}, true},
		{"missing one", []string{"a", "b"}, []string{"a", "c"}, false},
		{"empty need", []string{"a", "b"}, []string{}, true},
		{"nil need", []string{"a", "b"}, nil, true},
		{"empty scopes", []string{}, []string{"a"}, false},
		{"exact match", []string{"a", "b"}, []string{"a", "b"}, true},
		{"subset", []string{"a", "b", "c", "d"}, []string{"b", "d"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scope.HasAll(tt.scopes, tt.need...)
			if got != tt.want {
				t.Fatalf("HasAll(%v, %v) = %v, want %v", tt.scopes, tt.need, got, tt.want)
			}
		})
	}
}

func TestHasAny(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		scopes []string
		any    []string
		want   bool
	}{
		{"one match", []string{"a", "b", "c"}, []string{"x", "b", "y"}, true},
		{"no match", []string{"a", "b"}, []string{"x", "y"}, false},
		{"empty any", []string{"a", "b"}, []string{}, true},
		{"nil any", []string{"a", "b"}, nil, true},
		{"empty scopes", []string{}, []string{"a"}, false},
		{"all match", []string{"a", "b"}, []string{"a", "b"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scope.HasAny(tt.scopes, tt.any...)
			if got != tt.want {
				t.Fatalf("HasAny(%v, %v) = %v, want %v", tt.scopes, tt.any, got, tt.want)
			}
		})
	}
}

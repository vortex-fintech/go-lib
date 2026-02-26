package metadata_test

import (
	"context"
	"testing"

	"github.com/vortex-fintech/go-lib/transport/grpc/metadata"
	gmd "google.golang.org/grpc/metadata"
)

func TestWithBearer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		token     string
		wantEmpty bool
		wantValue string
	}{
		{"empty string", "", true, ""},
		{"whitespace only", "   ", true, ""},
		{"plain token", "mytoken", false, "Bearer mytoken"},
		{"token with bearer prefix", "Bearer mytoken", false, "Bearer mytoken"},
		{"token with BEARER prefix", "BEARER mytoken", false, "BEARER mytoken"},
		{"token with whitespace", "  mytoken  ", false, "Bearer mytoken"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := metadata.WithBearer(context.Background(), tt.token)
			if tt.wantEmpty {
				_, ok := gmd.FromOutgoingContext(ctx)
				if ok {
					t.Fatalf("expected no metadata, got some")
				}
				return
			}

			md, ok := gmd.FromOutgoingContext(ctx)
			if !ok {
				t.Fatalf("expected metadata, got none")
			}
			v := md.Get("authorization")
			if len(v) == 0 {
				t.Fatalf("expected authorization header")
			}
			if v[0] != tt.wantValue {
				t.Fatalf("got %q, want %q", v[0], tt.wantValue)
			}
		})
	}
}

func TestWithPoP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		x5tS256   string
		wantEmpty bool
	}{
		{"empty string", "", true},
		{"whitespace only", "   ", true},
		{"valid thumbprint", "abc123", false},
		{"thumbprint with whitespace", "  abc123  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := metadata.WithPoP(context.Background(), tt.x5tS256)
			if tt.wantEmpty {
				_, ok := gmd.FromOutgoingContext(ctx)
				if ok {
					t.Fatalf("expected no metadata, got some")
				}
				return
			}

			md, ok := gmd.FromOutgoingContext(ctx)
			if !ok {
				t.Fatalf("expected metadata, got none")
			}
			v := md.Get("x-pop")
			if len(v) == 0 {
				t.Fatalf("expected x-pop header")
			}
		})
	}
}

func TestWithAZP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		azp       string
		wantEmpty bool
	}{
		{"empty string", "", true},
		{"whitespace only", "   ", true},
		{"valid azp", "mobile-app", false},
		{"azp with whitespace", "  mobile-app  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := metadata.WithAZP(context.Background(), tt.azp)
			if tt.wantEmpty {
				_, ok := gmd.FromOutgoingContext(ctx)
				if ok {
					t.Fatalf("expected no metadata, got some")
				}
				return
			}

			md, ok := gmd.FromOutgoingContext(ctx)
			if !ok {
				t.Fatalf("expected metadata, got none")
			}
			v := md.Get("x-azp")
			if len(v) == 0 {
				t.Fatalf("expected x-azp header")
			}
		})
	}
}

func TestGet_Incoming(t *testing.T) {
	t.Parallel()

	ctx := gmd.NewIncomingContext(context.Background(), gmd.Pairs("authorization", "Bearer token123"))

	got := metadata.Get(ctx, "authorization")
	if got != "Bearer token123" {
		t.Fatalf("got %q, want %q", got, "Bearer token123")
	}
}

func TestGet_Outgoing(t *testing.T) {
	t.Parallel()

	ctx := gmd.NewOutgoingContext(context.Background(), gmd.Pairs("authorization", "Bearer token456"))

	got := metadata.Get(ctx, "authorization")
	if got != "Bearer token456" {
		t.Fatalf("got %q, want %q", got, "Bearer token456")
	}
}

func TestGet_IncomingPriority(t *testing.T) {
	t.Parallel()

	outgoing := gmd.NewOutgoingContext(context.Background(), gmd.Pairs("authorization", "outgoing"))
	ctx := gmd.NewIncomingContext(outgoing, gmd.Pairs("authorization", "incoming"))

	got := metadata.Get(ctx, "authorization")
	if got != "incoming" {
		t.Fatalf("got %q, want %q (incoming should have priority)", got, "incoming")
	}
}

func TestGet_Missing(t *testing.T) {
	t.Parallel()

	got := metadata.Get(context.Background(), "authorization")
	if got != "" {
		t.Fatalf("got %q, want empty string", got)
	}
}

func TestGetAll_Incoming(t *testing.T) {
	t.Parallel()

	ctx := gmd.NewIncomingContext(context.Background(), gmd.Pairs(
		"x-custom", "value1",
		"x-custom", "value2",
	))

	got := metadata.GetAll(ctx, "x-custom")
	if len(got) != 2 {
		t.Fatalf("got %d values, want 2", len(got))
	}
	if got[0] != "value1" || got[1] != "value2" {
		t.Fatalf("got %v, want [value1, value2]", got)
	}
}

func TestGetAll_Missing(t *testing.T) {
	t.Parallel()

	got := metadata.GetAll(context.Background(), "x-missing")
	if got != nil {
		t.Fatalf("got %v, want nil", got)
	}
}

func TestMergeOutgoing_PreservesExisting(t *testing.T) {
	t.Parallel()

	ctx := gmd.NewOutgoingContext(context.Background(), gmd.Pairs("x-existing", "kept"))
	ctx = metadata.WithBearer(ctx, "token123")

	md, ok := gmd.FromOutgoingContext(ctx)
	if !ok {
		t.Fatalf("expected metadata, got none")
	}

	existing := md.Get("x-existing")
	if len(existing) == 0 || existing[0] != "kept" {
		t.Fatalf("x-existing: got %v, want 'kept'", existing)
	}

	auth := md.Get("authorization")
	if len(auth) == 0 || auth[0] != "Bearer token123" {
		t.Fatalf("authorization: got %v, want 'Bearer token123'", auth)
	}
}

func TestMergeOutgoing_Overwrites(t *testing.T) {
	t.Parallel()

	ctx := gmd.NewOutgoingContext(context.Background(), gmd.Pairs("authorization", "Bearer old"))
	ctx = metadata.WithBearer(ctx, "new")

	md, _ := gmd.FromOutgoingContext(ctx)
	auth := md.Get("authorization")
	if len(auth) == 0 || auth[0] != "Bearer new" {
		t.Fatalf("got %v, want 'Bearer new'", auth)
	}
}

func TestConstants(t *testing.T) {
	t.Parallel()

	if metadata.HeaderAuthorization != "authorization" {
		t.Fatalf("HeaderAuthorization: got %q", metadata.HeaderAuthorization)
	}
	if metadata.HeaderPoP != "x-pop" {
		t.Fatalf("HeaderPoP: got %q", metadata.HeaderPoP)
	}
	if metadata.HeaderAZP != "x-azp" {
		t.Fatalf("HeaderAZP: got %q", metadata.HeaderAZP)
	}
}

func TestWithHelpers_NilContext(t *testing.T) {
	t.Parallel()

	ctx := metadata.WithBearer(nil, "token")
	if got := metadata.Get(ctx, "authorization"); got != "Bearer token" {
		t.Fatalf("unexpected authorization: %q", got)
	}

	ctx = metadata.WithPoP(nil, "thumb")
	if got := metadata.Get(ctx, "x-pop"); got != "thumb" {
		t.Fatalf("unexpected x-pop: %q", got)
	}

	ctx = metadata.WithAZP(nil, "mobile-app")
	if got := metadata.Get(ctx, "x-azp"); got != "mobile-app" {
		t.Fatalf("unexpected x-azp: %q", got)
	}
}

func TestGetAndGetAll_NilContext(t *testing.T) {
	t.Parallel()

	if got := metadata.Get(nil, "authorization"); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
	if got := metadata.GetAll(nil, "authorization"); got != nil {
		t.Fatalf("expected nil slice, got %v", got)
	}
}

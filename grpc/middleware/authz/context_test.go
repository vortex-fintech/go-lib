package authz

import (
	"context"
	"testing"
)

func TestIdentityRoundtrip(t *testing.T) {
	id := Identity{UserID: "user:1", Scopes: []string{"wallet:read"}, SID: "sess:1"}
	ctx := WithIdentity(context.Background(), id)
	got, ok := IdentityFrom(ctx)
	if !ok {
		t.Fatalf("expected ok")
	}
	if got.UserID != id.UserID || got.SID != id.SID {
		t.Fatalf("mismatch: %#v", got)
	}
	if len(got.Scopes) != 1 || got.Scopes[0] != "wallet:read" {
		t.Fatalf("bad scopes: %#v", got.Scopes)
	}
}

package authz

import (
	"context"
	"errors"
	"testing"

	libjwt "github.com/vortex-fintech/go-lib/security/jwt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type okVerifier struct{ cl *libjwt.Claims }

func (v okVerifier) Verify(ctx context.Context, raw string) (*libjwt.Claims, error) { return v.cl, nil }

type errVerifier struct{}

func (errVerifier) Verify(ctx context.Context, raw string) (*libjwt.Claims, error) {
	return nil, errors.New("bad token")
}

func mdCtx(token string) context.Context {
	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	return metadata.NewIncomingContext(context.Background(), md)
}

func TestUnary_OK(t *testing.T) {
	cfg := Config{
		Verifier:       okVerifier{cl: &libjwt.Claims{Subject: "user:1", Audience: []string{"wallet"}, Scopes: []string{"wallet:read"}, Sid: "sess:1"}},
		Audience:       "wallet",
		RequiredScopes: []string{"wallet:read"},
	}
	ic := UnaryServerInterceptor(cfg)
	ctx := mdCtx("any")

	called := false
	h := func(ctx context.Context, req any) (any, error) {
		called = true
		id, ok := IdentityFrom(ctx)
		if !ok || id.UserID != "user:1" || id.SID != "sess:1" || len(id.Scopes) != 1 {
			t.Fatalf("bad identity: %#v ok=%v", id, ok)
		}
		return "ok", nil
	}
	resp, err := ic(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, h)
	if err != nil || !called || resp != "ok" {
		t.Fatalf("want ok, got resp=%v err=%v called=%v", resp, err, called)
	}
}

func TestUnary_InvalidToken(t *testing.T) {
	cfg := Config{Verifier: errVerifier{}}
	ic := UnaryServerInterceptor(cfg)
	_, err := ic(mdCtx("x"), nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req any) (any, error) { return nil, nil })
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("want Unauthenticated, got %v", err)
	}
}

func TestUnary_AudienceMismatch(t *testing.T) {
	cfg := Config{
		Verifier: okVerifier{cl: &libjwt.Claims{Audience: []string{"other"}}},
		Audience: "wallet",
	}
	ic := UnaryServerInterceptor(cfg)
	_, err := ic(mdCtx("x"), nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req any) (any, error) { return nil, nil })
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("want PermissionDenied, got %v", err)
	}
}

func TestUnary_InsufficientScope(t *testing.T) {
	cfg := Config{
		Verifier:       okVerifier{cl: &libjwt.Claims{Audience: []string{"wallet"}, Scopes: []string{"wallet:read"}}},
		Audience:       "wallet",
		RequiredScopes: []string{"wallet:transfer:create"},
	}
	ic := UnaryServerInterceptor(cfg)
	_, err := ic(mdCtx("x"), nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req any) (any, error) { return nil, nil })
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("want PermissionDenied, got %v", err)
	}
}

func TestUnary_MissingBearer(t *testing.T) {
	cfg := Config{Verifier: okVerifier{cl: &libjwt.Claims{}}}
	ic := UnaryServerInterceptor(cfg)
	_, err := ic(context.Background(), nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req any) (any, error) { return nil, nil })
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("want Unauthenticated, got %v", err)
	}
}

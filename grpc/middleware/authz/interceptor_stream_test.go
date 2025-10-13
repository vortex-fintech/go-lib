package authz

import (
	"context"
	"testing"

	libjwt "github.com/vortex-fintech/go-lib/security/jwt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type dummyStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (d dummyStream) Context() context.Context { return d.ctx }

func mdStreamCtx() context.Context {
	md := metadata.New(map[string]string{"authorization": "Bearer y"})
	return metadata.NewIncomingContext(context.Background(), md)
}

func TestStream_OK(t *testing.T) {
	cfg := Config{
		Verifier:       okVerifier{cl: &libjwt.Claims{Subject: "user:1", Audience: []string{"wallet"}, Scopes: []string{"wallet:read"}, Sid: "sess:1"}},
		Audience:       "wallet",
		RequiredScopes: []string{"wallet:read"},
	}
	ic := StreamServerInterceptor(cfg)

	called := false
	stream := dummyStream{ctx: mdStreamCtx()}
	err := ic(nil, stream, &grpc.StreamServerInfo{FullMethod: "/svc/Stream"}, func(srv interface{}, ss grpc.ServerStream) error {
		called = true
		id, ok := IdentityFrom(ss.Context())
		if !ok || id.UserID != "user:1" {
			return status.Error(codes.Internal, "identity missing")
		}
		return nil
	})
	if err != nil || !called {
		t.Fatalf("want ok, got err=%v called=%v", err, called)
	}
}

func TestStream_InsufficientScope(t *testing.T) {
	cfg := Config{
		Verifier:       okVerifier{cl: &libjwt.Claims{Audience: []string{"wallet"}, Scopes: []string{"wallet:read"}}},
		Audience:       "wallet",
		RequiredScopes: []string{"wallet:transfer:create"},
	}
	ic := StreamServerInterceptor(cfg)
	stream := dummyStream{ctx: mdStreamCtx()}
	err := ic(nil, stream, &grpc.StreamServerInfo{FullMethod: "/svc/Stream"}, func(srv interface{}, ss grpc.ServerStream) error { return nil })
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("want PermissionDenied, got %v", err)
	}
}

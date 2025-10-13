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

func mdCtx2() context.Context {
	md := metadata.New(map[string]string{"authorization": "Bearer x"})
	return metadata.NewIncomingContext(context.Background(), md)
}

func TestUnary_Policy_Any_All_OK(t *testing.T) {
	cfg := Config{
		Verifier: okVerifier{cl: &libjwt.Claims{
			Subject:  "user:1",
			Audience: []string{"wallet"},
			Scopes:   []string{"wallet:access", "wallet:transfer:create"},
			Sid:      "sess:1",
		}},
		Audience: "wallet",
		ResolvePolicy: func(full string) Policy {
			switch full {
			case "/wallet.v1.WalletService/GetBalance":
				return Policy{Any: []string{"wallet:read", "wallet:access"}} // есть wallet:access
			case "/wallet.v1.WalletService/CreateTransfer":
				return Policy{All: []string{"wallet:transfer:create"}} // есть transfer:create
			default:
				return Policy{}
			}
		},
	}
	ic := UnaryServerInterceptor(cfg)

	// Any-политика
	if _, err := ic(mdCtx2(), nil, &grpc.UnaryServerInfo{FullMethod: "/wallet.v1.WalletService/GetBalance"}, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}); err != nil {
		t.Fatalf("ANY policy should pass, got %v", err)
	}

	// All-политика
	if _, err := ic(mdCtx2(), nil, &grpc.UnaryServerInfo{FullMethod: "/wallet.v1.WalletService/CreateTransfer"}, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}); err != nil {
		t.Fatalf("ALL policy should pass, got %v", err)
	}
}

func TestUnary_Policy_Any_Fail(t *testing.T) {
	cfg := Config{
		Verifier: okVerifier{cl: &libjwt.Claims{
			Audience: []string{"wallet"},
			Scopes:   []string{"wallet:read"},
		}},
		Audience: "wallet",
		ResolvePolicy: func(full string) Policy {
			return Policy{Any: []string{"wallet:access", "wallet:admin"}}
		},
	}
	ic := UnaryServerInterceptor(cfg)
	_, err := ic(mdCtx2(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, func(ctx context.Context, req any) (any, error) { return nil, nil })
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("want PermissionDenied for ANY fail, got %v", err)
	}
}

func TestUnary_Policy_All_Fail(t *testing.T) {
	cfg := Config{
		Verifier: okVerifier{cl: &libjwt.Claims{
			Audience: []string{"wallet"},
			Scopes:   []string{"wallet:read"},
		}},
		Audience: "wallet",
		ResolvePolicy: func(full string) Policy {
			return Policy{All: []string{"wallet:read", "wallet:transfer:create"}}
		},
	}
	ic := UnaryServerInterceptor(cfg)
	_, err := ic(mdCtx2(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, func(ctx context.Context, req any) (any, error) { return nil, nil })
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("want PermissionDenied for ALL fail, got %v", err)
	}
}

func TestUnary_GlobalRequiredScopes_And_MethodAny(t *testing.T) {
	cfg := Config{
		Verifier: okVerifier{cl: &libjwt.Claims{
			Audience: []string{"wallet"},
			Scopes:   []string{"wallet:base", "wallet:read"},
		}},
		Audience:       "wallet",
		RequiredScopes: []string{"wallet:base"}, // глобальный ALL
		ResolvePolicy: func(full string) Policy {
			return Policy{Any: []string{"wallet:read", "wallet:access"}} // есть wallet:read
		},
	}
	ic := UnaryServerInterceptor(cfg)
	if _, err := ic(mdCtx2(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, func(ctx context.Context, req any) (any, error) { return "ok", nil }); err != nil {
		t.Fatalf("global ALL + method ANY should pass, got %v", err)
	}
}

package authz

import (
	"context"
	"errors"
	"testing"
	"time"

	libjwt "github.com/vortex-fintech/go-lib/security/jwt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type verifierStub struct {
	claims *libjwt.Claims
	err    error
	called int
}

func (v *verifierStub) Verify(_ context.Context, _ string) (*libjwt.Claims, error) {
	v.called++
	if v.err != nil {
		return nil, v.err
	}
	return v.claims, nil
}

func TestValidateConfig_Invalid(t *testing.T) {
	t.Parallel()

	err := ValidateConfig(Config{})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got %v", err)
	}

	var cfgErr *ConfigValidationError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("expected ConfigValidationError, got %T", err)
	}
	if cfgErr.Field != "Verifier" {
		t.Fatalf("expected Verifier field, got %s", cfgErr.Field)
	}
}

func TestUnaryServerInterceptor_InvalidConfig_ReturnsInternalWithoutPanic(t *testing.T) {
	t.Parallel()

	interceptor := UnaryServerInterceptor(Config{})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	_, err := interceptor(context.Background(), struct{}{}, &grpc.UnaryServerInfo{FullMethod: "/svc.Method"}, passHandler)
	if err == nil {
		t.Fatalf("expected error")
	}
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal, got %v", status.Code(err))
	}
}

func TestStreamServerInterceptor_InvalidConfig_ReturnsInternalWithoutPanic(t *testing.T) {
	t.Parallel()

	interceptor := StreamServerInterceptor(Config{})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	err := interceptor(struct{}{}, &streamStub{ctx: context.Background()}, &grpc.StreamServerInfo{FullMethod: "/svc.Stream"}, func(srv any, stream grpc.ServerStream) error {
		return nil
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal, got %v", status.Code(err))
	}
}

func TestUnaryServerInterceptor_MissingMetadata(t *testing.T) {
	t.Parallel()

	v := &verifierStub{claims: validClaims("thumb")}
	interceptor := UnaryServerInterceptor(Config{
		Verifier:       v,
		Audience:       "wallet",
		Actor:          "api-gateway",
		RequireScopes:  true,
		RequirePoP:     true,
		MTLSThumbprint: func(context.Context) string { return "thumb" },
	})

	_, err := interceptor(context.Background(), struct{}{}, &grpc.UnaryServerInfo{FullMethod: "/svc.Method"}, passHandler)
	if err == nil {
		t.Fatalf("expected error")
	}
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", status.Code(err))
	}
}

func TestUnaryServerInterceptor_SkipAuth(t *testing.T) {
	t.Parallel()

	v := &verifierStub{claims: validClaims("thumb")}
	interceptor := UnaryServerInterceptor(Config{
		Verifier:   v,
		Audience:   "wallet",
		SkipAuth:   SliceSkipAuth("/svc.Public"),
		RequirePoP: false,
	})

	_, err := interceptor(context.Background(), struct{}{}, &grpc.UnaryServerInfo{FullMethod: "/svc.Public"}, passHandler)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if v.called != 0 {
		t.Fatalf("verifier should not be called on skip-auth path")
	}
}

func TestUnaryServerInterceptor_InsufficientScope(t *testing.T) {
	t.Parallel()

	v := &verifierStub{claims: validClaims("thumb")}
	interceptor := UnaryServerInterceptor(Config{
		Verifier:       v,
		Audience:       "wallet",
		Actor:          "api-gateway",
		RequireScopes:  true,
		RequirePoP:     true,
		MTLSThumbprint: func(context.Context) string { return "thumb" },
		RequiredScopes: []string{"admin:write"},
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer token"))
	_, err := interceptor(ctx, struct{}{}, &grpc.UnaryServerInfo{FullMethod: "/svc.Method"}, passHandler)
	if err == nil {
		t.Fatalf("expected error")
	}
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", status.Code(err))
	}
}

func TestUnaryServerInterceptor_SetsIdentityAndClaims(t *testing.T) {
	t.Parallel()

	v := &verifierStub{claims: validClaims("thumb")}
	interceptor := UnaryServerInterceptor(Config{
		Verifier:       v,
		Audience:       "wallet",
		Actor:          "api-gateway",
		RequireScopes:  true,
		RequirePoP:     true,
		MTLSThumbprint: func(context.Context) string { return "thumb" },
		ResolvePolicy: MapResolver(map[string]Policy{
			"/svc.Method": {All: []string{"wallet:read"}},
		}),
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer token"))
	_, err := interceptor(ctx, struct{}{}, &grpc.UnaryServerInfo{FullMethod: "/svc.Method"}, func(ctx context.Context, req any) (any, error) {
		id, ok := IdentityFrom(ctx)
		if !ok {
			t.Fatalf("identity missing in context")
		}
		if id.UserID.String() != "550e8400-e29b-41d4-a716-446655440000" {
			t.Fatalf("unexpected user id: %s", id.UserID)
		}
		cl, ok := ClaimsFrom(ctx)
		if !ok || cl == nil {
			t.Fatalf("claims missing in context")
		}
		if cl.WalletID != "w-1" {
			t.Fatalf("unexpected wallet id: %s", cl.WalletID)
		}
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnaryServerInterceptor_InvalidTokenMapsToUnauthenticated(t *testing.T) {
	t.Parallel()

	v := &verifierStub{err: errors.New("boom")}
	interceptor := UnaryServerInterceptor(Config{Verifier: v, Audience: "wallet"})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer token"))
	_, err := interceptor(ctx, struct{}{}, &grpc.UnaryServerInfo{FullMethod: "/svc.Method"}, passHandler)
	if err == nil {
		t.Fatalf("expected error")
	}
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", status.Code(err))
	}
}

func TestUnaryServerInterceptor_MissingPoP(t *testing.T) {
	t.Parallel()

	v := &verifierStub{claims: validClaims("thumb")}
	interceptor := UnaryServerInterceptor(Config{
		Verifier:       v,
		Audience:       "wallet",
		Actor:          "api-gateway",
		RequireScopes:  true,
		RequirePoP:     true,
		MTLSThumbprint: func(context.Context) string { return "" },
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer token"))
	_, err := interceptor(ctx, struct{}{}, &grpc.UnaryServerInfo{FullMethod: "/svc.Method"}, passHandler)
	if err == nil {
		t.Fatalf("expected error")
	}
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", status.Code(err))
	}
}

func TestUnaryServerInterceptor_RequirePoPDisabled_AllowsMissingPoP(t *testing.T) {
	t.Parallel()

	v := &verifierStub{claims: validClaims("thumb")}
	interceptor := UnaryServerInterceptor(Config{
		Verifier:       v,
		Audience:       "wallet",
		Actor:          "api-gateway",
		RequireScopes:  true,
		RequirePoP:     false,
		MTLSThumbprint: func(context.Context) string { return "" },
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer token"))
	_, err := interceptor(ctx, struct{}{}, &grpc.UnaryServerInfo{FullMethod: "/svc.Method"}, passHandler)
	if err != nil {
		t.Fatalf("expected no error when RequirePoP=false, got %v", err)
	}
}

func TestStreamServerInterceptor_SetsIdentityAndClaims(t *testing.T) {
	t.Parallel()

	v := &verifierStub{claims: validClaims("thumb")}
	interceptor := StreamServerInterceptor(Config{
		Verifier:       v,
		Audience:       "wallet",
		Actor:          "api-gateway",
		RequireScopes:  true,
		RequirePoP:     true,
		MTLSThumbprint: func(context.Context) string { return "thumb" },
		ResolvePolicy: MapResolver(map[string]Policy{
			"/svc.Stream": {Any: []string{"wallet:read"}},
		}),
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer token"))
	ss := &streamStub{ctx: ctx}

	err := interceptor(struct{}{}, ss, &grpc.StreamServerInfo{FullMethod: "/svc.Stream"}, func(srv any, stream grpc.ServerStream) error {
		id, ok := IdentityFrom(stream.Context())
		if !ok {
			t.Fatalf("identity missing in stream context")
		}
		if id.UserID.String() != "550e8400-e29b-41d4-a716-446655440000" {
			t.Fatalf("unexpected user id: %s", id.UserID)
		}
		cl, ok := ClaimsFrom(stream.Context())
		if !ok || cl == nil {
			t.Fatalf("claims missing in stream context")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStreamServerInterceptor_InsufficientScope(t *testing.T) {
	t.Parallel()

	v := &verifierStub{claims: validClaims("thumb")}
	interceptor := StreamServerInterceptor(Config{
		Verifier:       v,
		Audience:       "wallet",
		Actor:          "api-gateway",
		RequireScopes:  true,
		RequirePoP:     true,
		MTLSThumbprint: func(context.Context) string { return "thumb" },
		ResolvePolicy: MapResolver(map[string]Policy{
			"/svc.Stream": {All: []string{"admin:write"}},
		}),
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer token"))
	ss := &streamStub{ctx: ctx}

	err := interceptor(struct{}{}, ss, &grpc.StreamServerInfo{FullMethod: "/svc.Stream"}, func(srv any, stream grpc.ServerStream) error {
		return nil
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", status.Code(err))
	}
}

func validClaims(thumb string) *libjwt.Claims {
	now := time.Now()
	return &libjwt.Claims{
		Issuer:   "issuer",
		Subject:  "550e8400-e29b-41d4-a716-446655440000",
		Audience: []string{"wallet"},
		Iat:      now.Add(-1 * time.Minute).Unix(),
		Exp:      now.Add(2 * time.Minute).Unix(),
		Jti:      "jti-1",
		Scopes:   []string{"wallet:read", "payments:create"},
		Act:      &libjwt.Actor{Sub: "api-gateway"},
		Cnf:      &libjwt.Cnf{X5tS256: thumb},
		WalletID: "w-1",
	}
}

func passHandler(ctx context.Context, req any) (any, error) {
	return req, nil
}

type streamStub struct {
	ctx context.Context
}

func (s *streamStub) SetHeader(metadata.MD) error  { return nil }
func (s *streamStub) SendHeader(metadata.MD) error { return nil }
func (s *streamStub) SetTrailer(metadata.MD)       {}
func (s *streamStub) Context() context.Context     { return s.ctx }
func (s *streamStub) SendMsg(any) error            { return nil }
func (s *streamStub) RecvMsg(any) error            { return nil }

package authz

import (
	"context"
	"errors"
	"strings"

	scope "github.com/vortex-fintech/go-lib/authz/scope"
	libjwt "github.com/vortex-fintech/go-lib/security/jwt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Policy — требования по скоупам.
type Policy struct {
	All []string // все обязательны
	Any []string // хотя бы один обязателен
}

// PolicyResolver возвращает политику по полному имени метода.
type PolicyResolver func(fullMethod string) Policy

// SkipAuthFunc возвращает true, если метод анонимный (пропускать JWT-проверку).
type SkipAuthFunc func(fullMethod string) bool

type Config struct {
	Verifier       libjwt.Verifier
	Audience       string
	RequiredScopes []string
	CheckAudience  libjwt.AudienceChecker
	ResolvePolicy  PolicyResolver
	SkipAuth       SkipAuthFunc
}

func UnaryServerInterceptor(cfg Config) grpc.UnaryServerInterceptor {
	if cfg.CheckAudience == nil {
		cfg.CheckAudience = libjwt.DefaultAudienceChecker
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// анонимные методы
		if cfg.SkipAuth != nil && cfg.SkipAuth(info.FullMethod) {
			return handler(ctx, req)
		}

		// jwt + aud + scopes
		raw, err := bearerFromMD(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}
		cl, err := cfg.Verifier.Verify(ctx, raw)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}
		if cfg.Audience != "" && !cfg.CheckAudience(cl, cfg.Audience) {
			return nil, status.Error(codes.PermissionDenied, "audience mismatch")
		}

		var p Policy
		if cfg.ResolvePolicy != nil {
			p = cfg.ResolvePolicy(info.FullMethod)
		}
		if !satisfies(cl.Scopes, p, cfg.RequiredScopes) {
			return nil, status.Error(codes.PermissionDenied, "insufficient scope")
		}

		id := Identity{UserID: cl.Subject, Scopes: cl.Scopes, SID: cl.Sid}
		ctx = WithIdentity(ctx, id)
		return handler(ctx, req)
	}
}

func StreamServerInterceptor(cfg Config) grpc.StreamServerInterceptor {
	if cfg.CheckAudience == nil {
		cfg.CheckAudience = libjwt.DefaultAudienceChecker
	}
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// анонимные методы
		if cfg.SkipAuth != nil && cfg.SkipAuth(info.FullMethod) {
			return handler(srv, ss)
		}

		// jwt + aud + scopes
		ctx := ss.Context()
		raw, err := bearerFromMD(ctx)
		if err != nil {
			return status.Error(codes.Unauthenticated, err.Error())
		}
		cl, err := cfg.Verifier.Verify(ctx, raw)
		if err != nil {
			return status.Error(codes.Unauthenticated, "invalid token")
		}
		if cfg.Audience != "" && !cfg.CheckAudience(cl, cfg.Audience) {
			return status.Error(codes.PermissionDenied, "audience mismatch")
		}

		var p Policy
		if cfg.ResolvePolicy != nil {
			p = cfg.ResolvePolicy(info.FullMethod)
		}
		if !satisfies(cl.Scopes, p, cfg.RequiredScopes) {
			return status.Error(codes.PermissionDenied, "insufficient scope")
		}

		id := Identity{UserID: cl.Subject, Scopes: cl.Scopes, SID: cl.Sid}
		wrapped := &serverStream{ServerStream: ss, ctx: WithIdentity(ctx, id)}
		return handler(srv, wrapped)
	}
}

type serverStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *serverStream) Context() context.Context { return s.ctx }

func satisfies(have []string, p Policy, globalAll []string) bool {
	if len(globalAll) > 0 && !scope.HasAll(have, globalAll...) {
		return false
	}
	if len(p.All) > 0 && !scope.HasAll(have, p.All...) {
		return false
	}
	if len(p.Any) > 0 && !scope.HasAny(have, p.Any...) {
		return false
	}
	return true
}

func bearerFromMD(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("missing metadata")
	}
	vals := md.Get("authorization")
	if len(vals) == 0 {
		return "", errors.New("missing authorization")
	}
	parts := strings.SplitN(vals[0], " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("invalid authorization")
	}
	return parts[1], nil
}

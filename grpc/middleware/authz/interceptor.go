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

// Policy описывает требования по скоупам:
// - All: все перечисленные обязательны
// - Any: хотя бы один из перечисленных обязателен
type Policy struct {
	All []string
	Any []string
}

// PolicyResolver возвращает политику по полному имени метода.
type PolicyResolver func(fullMethod string) Policy

type Config struct {
	Verifier       libjwt.Verifier
	Audience       string   // требуемая аудитория для сервиса
	RequiredScopes []string // глобальные "ALL" для всего сервиса (опционально)
	CheckAudience  libjwt.AudienceChecker
	ResolvePolicy  PolicyResolver // политика на метод (опционально)
}

func UnaryServerInterceptor(cfg Config) grpc.UnaryServerInterceptor {
	if cfg.CheckAudience == nil {
		cfg.CheckAudience = libjwt.DefaultAudienceChecker
	}
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
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
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
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

// satisfies применяет глобальные "ALL" и методовые "ALL"/"ANY".
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

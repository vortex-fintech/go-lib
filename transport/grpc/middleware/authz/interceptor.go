// go-lib/authz/interceptor.go
package authz

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	scope "github.com/vortex-fintech/go-lib/security/scope"
	libjwt "github.com/vortex-fintech/go-lib/security/jwt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// Policy — требования по скоупам.
type Policy struct {
	All []string // все обязательны
	Any []string // хотя бы один обязателен
}

// PolicyResolver — политика по полному имени метода.
type PolicyResolver func(fullMethod string) Policy

// SkipAuthFunc — пропустить аутентификацию для метода.
type SkipAuthFunc func(fullMethod string) bool

type Config struct {
	Verifier libjwt.Verifier

	// OBO-политика (ValidateOBO)
	Audience       string                           // этот сервис, напр. "wallet" (обязателен)
	Actor          string                           // кто обменял токен, напр. "api-gateway" (желателен)
	AllowedAZP     []string                         // допустимые источники клиента (azp), опционально (если задано — azp обязателен)
	Leeway         time.Duration                    // 30–60s
	MaxTTL         time.Duration                    // <= 5m
	RequireScopes  bool                             // требовать непустые scopes
	SeenJTI        func(string) bool                // anti-replay (может быть nil)
	RequirePoP     bool                             // требовать PoP (x5t#S256)
	MTLSThumbprint func(ctx context.Context) string // вернуть x5t#S256 из peer TLS (может быть nil)

	// Доп. авторизация на уровне метода
	RequiredScopes []string
	ResolvePolicy  PolicyResolver

	// Анонимные методы
	SkipAuth SkipAuthFunc
}

func UnaryServerInterceptor(cfg Config) grpc.UnaryServerInterceptor {
	cfg = normalize(cfg)
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if cfg.SkipAuth != nil && cfg.SkipAuth(info.FullMethod) {
			return handler(ctx, req)
		}

		raw, err := bearerFromMD(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}
		cl, err := cfg.Verifier.Verify(ctx, raw)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		var thumb string
		if cfg.MTLSThumbprint != nil {
			thumb = cfg.MTLSThumbprint(ctx)
		}
		if cfg.RequirePoP && thumb == "" {
			return nil, status.Error(codes.Unauthenticated, "missing mTLS client certificate")
		}

		if err := libjwt.ValidateOBO(time.Now(), cl, libjwt.OBOValidateOptions{
			WantAudience:   cfg.Audience,
			WantActor:      cfg.Actor,
			AllowedAZP:     cfg.AllowedAZP,
			Leeway:         cfg.Leeway,
			MaxTTL:         cfg.MaxTTL,
			MTLSThumbprint: thumb,
			SeenJTI:        cfg.SeenJTI,
			RequireScopes:  cfg.RequireScopes,
		}); err != nil {
			switch err {
			case libjwt.ErrExpired, libjwt.ErrIATInFuture:
				return nil, status.Error(codes.Unauthenticated, err.Error())
			default:
				return nil, status.Error(codes.PermissionDenied, err.Error())
			}
		}

		uid, _ := uuid.Parse(cl.Subject)
		sc := cl.EffectiveScopes()

		var p Policy
		if cfg.ResolvePolicy != nil {
			p = cfg.ResolvePolicy(info.FullMethod)
		}
		if !satisfies(sc, p, cfg.RequiredScopes) {
			return nil, status.Error(codes.PermissionDenied, "insufficient scope")
		}

		id := Identity{UserID: uid, Scopes: sc, SID: cl.Sid, DeviceID: cl.DeviceID}
		ctx = WithIdentity(ctx, id)
		ctx = WithClaims(ctx, cl) // если где-то нужно читать wallet_id/azp/acr

		return handler(ctx, req)
	}
}

func StreamServerInterceptor(cfg Config) grpc.StreamServerInterceptor {
	cfg = normalize(cfg)
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if cfg.SkipAuth != nil && cfg.SkipAuth(info.FullMethod) {
			return handler(srv, ss)
		}

		ctx := ss.Context()
		raw, err := bearerFromMD(ctx)
		if err != nil {
			return status.Error(codes.Unauthenticated, err.Error())
		}
		cl, err := cfg.Verifier.Verify(ctx, raw)
		if err != nil {
			return status.Error(codes.Unauthenticated, "invalid token")
		}

		var thumb string
		if cfg.MTLSThumbprint != nil {
			thumb = cfg.MTLSThumbprint(ctx)
		}
		if cfg.RequirePoP && thumb == "" {
			return status.Error(codes.Unauthenticated, "missing mTLS client certificate")
		}

		if err := libjwt.ValidateOBO(time.Now(), cl, libjwt.OBOValidateOptions{
			WantAudience:   cfg.Audience,
			WantActor:      cfg.Actor,
			AllowedAZP:     cfg.AllowedAZP,
			Leeway:         cfg.Leeway,
			MaxTTL:         cfg.MaxTTL,
			MTLSThumbprint: thumb,
			SeenJTI:        cfg.SeenJTI,
			RequireScopes:  cfg.RequireScopes,
		}); err != nil {
			switch err {
			case libjwt.ErrExpired, libjwt.ErrIATInFuture:
				return status.Error(codes.Unauthenticated, err.Error())
			default:
				return status.Error(codes.PermissionDenied, err.Error())
			}
		}

		uid, _ := uuid.Parse(cl.Subject)
		sc := cl.EffectiveScopes()

		var p Policy
		if cfg.ResolvePolicy != nil {
			p = cfg.ResolvePolicy(info.FullMethod)
		}
		if !satisfies(sc, p, cfg.RequiredScopes) {
			return status.Error(codes.PermissionDenied, "insufficient scope")
		}

		id := Identity{UserID: uid, Scopes: sc, SID: cl.Sid, DeviceID: cl.DeviceID}
		wrapped := &serverStream{ServerStream: ss, ctx: WithClaims(WithIdentity(ctx, id), cl)}
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
		vals = md.Get("grpcgateway-authorization")
	}
	if len(vals) == 0 {
		return "", errors.New("missing authorization")
	}
	parts := strings.SplitN(strings.TrimSpace(vals[0]), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("invalid authorization")
	}
	return parts[1], nil
}

// mtlsThumbFromPeer — извлечь x5t#S256 из peer TLS.
func mtlsThumbFromPeer(ctx context.Context) string {
	pr, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}
	if ti, ok := pr.AuthInfo.(credentials.TLSInfo); ok {
		if len(ti.State.PeerCertificates) > 0 {
			return libjwt.X5tS256FromCert(ti.State.PeerCertificates[0])
		}
	}
	return ""
}

func normalize(cfg Config) Config {
	if cfg.Verifier == nil {
		panic("authz: Verifier must be set")
	}
	if cfg.Audience == "" {
		panic("authz: Audience must be set")
	}
	if cfg.Leeway <= 0 {
		cfg.Leeway = 45 * time.Second
	}
	if cfg.MaxTTL <= 0 {
		cfg.MaxTTL = 5 * time.Minute
	}
	if cfg.MTLSThumbprint == nil {
		cfg.MTLSThumbprint = mtlsThumbFromPeer
	}
	// Fail-safe: требуем PoP по умолчанию
	if !cfg.RequirePoP {
		cfg.RequirePoP = true
	}
	return cfg
}

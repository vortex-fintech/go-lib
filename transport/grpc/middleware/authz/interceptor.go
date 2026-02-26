package authz

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	libjwt "github.com/vortex-fintech/go-lib/security/jwt"
	scope "github.com/vortex-fintech/go-lib/security/scope"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type Policy struct {
	All []string
	Any []string
}

type PolicyResolver func(fullMethod string) Policy

type SkipAuthFunc func(fullMethod string) bool

type Config struct {
	Verifier libjwt.Verifier

	Audience       string
	Actor          string
	AllowedAZP     []string
	Leeway         time.Duration
	MaxTTL         time.Duration
	RequireScopes  bool
	SeenJTI        func(string) bool
	RequirePoP     bool
	MTLSThumbprint func(ctx context.Context) string

	RequiredScopes []string
	ResolvePolicy  PolicyResolver

	SkipAuth SkipAuthFunc
}

type AuthzResult struct {
	Identity Identity
	Claims   *libjwt.Claims
}

var ErrInvalidConfig = errors.New("authz: invalid config")

type ConfigValidationError struct {
	Field string
	Err   error
}

func (e *ConfigValidationError) Error() string {
	if e == nil {
		return ErrInvalidConfig.Error()
	}
	if e.Err == nil {
		return ErrInvalidConfig.Error() + ": " + e.Field
	}
	return ErrInvalidConfig.Error() + ": " + e.Field + ": " + e.Err.Error()
}

func (e *ConfigValidationError) Unwrap() error {
	if e == nil {
		return nil
	}
	if e.Err != nil {
		return errors.Join(ErrInvalidConfig, e.Err)
	}
	return ErrInvalidConfig
}

func ValidateConfig(cfg Config) error {
	if cfg.Verifier == nil {
		return &ConfigValidationError{Field: "Verifier", Err: errors.New("must be set")}
	}
	if cfg.Audience == "" {
		return &ConfigValidationError{Field: "Audience", Err: errors.New("must be set")}
	}
	return nil
}

func Authorize(ctx context.Context, fullMethod string, cfg Config) (*AuthzResult, error) {
	if err := ValidateConfig(cfg); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	cfg = normalize(cfg)

	if cfg.SkipAuth != nil && cfg.SkipAuth(fullMethod) {
		return nil, nil
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

	uid, err := uuid.Parse(cl.Subject)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, libjwt.ErrBadSubject.Error())
	}

	sc := cl.EffectiveScopes()

	var p Policy
	if cfg.ResolvePolicy != nil {
		p = cfg.ResolvePolicy(fullMethod)
	}
	if !satisfies(sc, p, cfg.RequiredScopes) {
		return nil, status.Error(codes.PermissionDenied, "insufficient scope")
	}

	return &AuthzResult{
		Identity: Identity{UserID: uid, Scopes: sc, SID: cl.Sid, DeviceID: cl.DeviceID},
		Claims:   cl,
	}, nil
}

func UnaryServerInterceptor(cfg Config) grpc.UnaryServerInterceptor {
	err := ValidateConfig(cfg)
	if err == nil {
		cfg = normalize(cfg)
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		result, err := Authorize(ctx, info.FullMethod, cfg)
		if err != nil {
			return nil, err
		}
		if result != nil {
			ctx = WithIdentity(ctx, result.Identity)
			ctx = WithClaims(ctx, result.Claims)
		}
		return handler(ctx, req)
	}
}

func StreamServerInterceptor(cfg Config) grpc.StreamServerInterceptor {
	err := ValidateConfig(cfg)
	if err == nil {
		cfg = normalize(cfg)
	}
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err != nil {
			return status.Error(codes.Internal, err.Error())
		}
		result, err := Authorize(ss.Context(), info.FullMethod, cfg)
		if err != nil {
			return err
		}
		if result != nil {
			ctx := WithClaims(WithIdentity(ss.Context(), result.Identity), result.Claims)
			wrapped := &serverStream{ServerStream: ss, ctx: ctx}
			return handler(srv, wrapped)
		}
		return handler(srv, ss)
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

func MTLSThumbprintFromPeer(ctx context.Context) string {
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
	if cfg.Leeway <= 0 {
		cfg.Leeway = 45 * time.Second
	}
	if cfg.MaxTTL <= 0 {
		cfg.MaxTTL = 5 * time.Minute
	}
	if cfg.MTLSThumbprint == nil {
		cfg.MTLSThumbprint = MTLSThumbprintFromPeer
	}
	return cfg
}

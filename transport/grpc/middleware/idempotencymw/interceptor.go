package idempotencymw

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const defaultHeader = "idempotency-key"

type Metadata struct {
	Principal      string
	GRPCMethod     string
	IdempotencyKey string
	RequestHash    string
}

type Config struct {
	RequireKey       bool
	Header           string
	MaxKeyLength     int
	IsMethodEnabled  func(fullMethod string) bool
	ResolvePrincipal func(ctx context.Context, md metadata.MD) string
}

type ctxKey struct{}

func Unary(cfg Config) grpc.UnaryServerInterceptor {
	header := strings.TrimSpace(cfg.Header)
	if header == "" {
		header = defaultHeader
	}
	maxLen := cfg.MaxKeyLength
	if maxLen <= 0 {
		maxLen = 128
	}
	enabled := cfg.IsMethodEnabled
	if enabled == nil {
		enabled = func(string) bool { return true }
	}
	resolve := cfg.ResolvePrincipal
	if resolve == nil {
		resolve = func(context.Context, metadata.MD) string { return "unknown" }
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !enabled(info.FullMethod) {
			return handler(ctx, req)
		}

		md, _ := metadata.FromIncomingContext(ctx)
		key := strings.TrimSpace(firstMD(md, header))
		if key == "" {
			if cfg.RequireKey {
				return nil, status.Error(codes.InvalidArgument, header+" is required for this method")
			}
			return handler(ctx, req)
		}
		if len(key) > maxLen {
			return nil, status.Error(codes.InvalidArgument, header+" is too long")
		}

		msg, ok := req.(proto.Message)
		if !ok {
			return nil, status.Error(codes.Internal, "request is not a protobuf message")
		}
		bytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(msg)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to hash request payload")
		}
		h := sha256.Sum256(bytes)

		ctx = context.WithValue(ctx, ctxKey{}, Metadata{
			Principal:      resolve(ctx, md),
			GRPCMethod:     info.FullMethod,
			IdempotencyKey: key,
			RequestHash:    hex.EncodeToString(h[:]),
		})
		return handler(ctx, req)
	}
}

func FromContext(ctx context.Context) (Metadata, bool) {
	v := ctx.Value(ctxKey{})
	m, ok := v.(Metadata)
	return m, ok
}

func firstMD(md metadata.MD, key string) string {
	vals := md.Get(key)
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}

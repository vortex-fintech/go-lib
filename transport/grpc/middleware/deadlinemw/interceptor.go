package deadlinemw

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

type Config struct {
	DefaultTimeout time.Duration
	MaxTimeout     time.Duration
	MethodTimeouts map[string]time.Duration
}

func Unary(cfg Config) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		wanted := cfg.DefaultTimeout
		if cfg.MethodTimeouts != nil {
			if v, ok := cfg.MethodTimeouts[info.FullMethod]; ok {
				wanted = v
			}
		}

		if dl, ok := ctx.Deadline(); ok {
			remaining := time.Until(dl)
			limit := wanted
			if cfg.MaxTimeout > 0 && (limit <= 0 || limit > cfg.MaxTimeout) {
				limit = cfg.MaxTimeout
			}
			if limit > 0 && remaining > limit {
				nctx, cancel := context.WithTimeout(ctx, limit)
				defer cancel()
				return handler(nctx, req)
			}
			return handler(ctx, req)
		}

		apply := wanted
		if cfg.MaxTimeout > 0 && (apply <= 0 || apply > cfg.MaxTimeout) {
			apply = cfg.MaxTimeout
		}
		if apply > 0 {
			nctx, cancel := context.WithTimeout(ctx, apply)
			defer cancel()
			return handler(nctx, req)
		}
		return handler(ctx, req)
	}
}

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

		apply := func(d time.Duration) (context.Context, context.CancelFunc) {
			if d <= 0 {
				return ctx, func() {}
			}
			return context.WithTimeout(ctx, d)
		}

		if dl, ok := ctx.Deadline(); ok {
			remaining := time.Until(dl)
			if cfg.MaxTimeout > 0 && remaining > cfg.MaxTimeout {
				nctx, cancel := apply(cfg.MaxTimeout)
				defer cancel()
				return handler(nctx, req)
			}
			if wanted > 0 && remaining > wanted {
				nctx, cancel := apply(wanted)
				defer cancel()
				return handler(nctx, req)
			}
			return handler(ctx, req)
		}

		if wanted > 0 {
			nctx, cancel := apply(wanted)
			defer cancel()
			return handler(nctx, req)
		}
		if cfg.MaxTimeout > 0 {
			nctx, cancel := apply(cfg.MaxTimeout)
			defer cancel()
			return handler(nctx, req)
		}
		return handler(ctx, req)
	}
}

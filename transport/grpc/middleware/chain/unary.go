package chain

import (
	cb "github.com/vortex-fintech/go-lib/transport/grpc/middleware/circuitbreaker"
	ctxcancel "github.com/vortex-fintech/go-lib/transport/grpc/middleware/contextcancel"
	errorsmw "github.com/vortex-fintech/go-lib/transport/grpc/middleware/errorsmw"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type Options struct {
	Pre  []grpc.UnaryServerInterceptor
	Post []grpc.UnaryServerInterceptor

	AuthzInterceptor grpc.UnaryServerInterceptor
	CircuitBreaker   *cb.Interceptor
	DisableCtxCancel bool
	DisableErrors    bool
}

type StreamOptions struct {
	Pre  []grpc.StreamServerInterceptor
	Post []grpc.StreamServerInterceptor

	AuthzInterceptor grpc.StreamServerInterceptor
	DisableCtxCancel bool
	DisableErrors    bool
}

func Default(opts Options) grpc.ServerOption {
	var chain []grpc.UnaryServerInterceptor

	if len(opts.Pre) > 0 {
		chain = append(chain, opts.Pre...)
	}

	if !opts.DisableCtxCancel {
		chain = append(chain, ctxcancel.Unary())
	}

	if opts.AuthzInterceptor != nil {
		chain = append(chain, opts.AuthzInterceptor)
	}

	if opts.CircuitBreaker != nil {
		chain = append(chain, opts.CircuitBreaker.Unary())
	}

	if !opts.DisableErrors {
		chain = append(chain, errorsmw.Unary())
	}

	if len(opts.Post) > 0 {
		chain = append(chain, opts.Post...)
	}

	return grpc.ChainUnaryInterceptor(chain...)
}

func DefaultStream(opts StreamOptions) grpc.ServerOption {
	var chain []grpc.StreamServerInterceptor

	if len(opts.Pre) > 0 {
		chain = append(chain, opts.Pre...)
	}

	if !opts.DisableCtxCancel {
		chain = append(chain, streamCtxCancel)
	}

	if opts.AuthzInterceptor != nil {
		chain = append(chain, opts.AuthzInterceptor)
	}

	if !opts.DisableErrors {
		chain = append(chain, errorsmw.Stream())
	}

	if len(opts.Post) > 0 {
		chain = append(chain, opts.Post...)
	}

	return grpc.ChainStreamInterceptor(chain...)
}

func streamCtxCancel(srv any, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if err := ss.Context().Err(); err != nil {
		return status.FromContextError(err).Err()
	}
	return handler(srv, ss)
}

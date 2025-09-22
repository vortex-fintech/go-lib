package errorsmw

import (
	"context"
	"errors"

	gliberrors "github.com/vortex-fintech/go-lib/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type grpcConvertible interface {
	ToGRPC() error
}

type Options struct {
	Fallback func(err error) error
}

type Option func(*Options)

func WithFallback(f func(error) error) Option {
	return func(o *Options) { o.Fallback = f }
}

func Unary(opts ...Option) grpc.UnaryServerInterceptor {
	o := Options{
		Fallback: func(_ error) error { return gliberrors.InternalError.ToGRPC() },
	}
	for _, f := range opts {
		f(&o)
	}

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {

		resp, err := handler(ctx, req)
		if err == nil {
			return resp, nil
		}

		if _, ok := status.FromError(err); ok {
			return nil, err
		}

		var conv grpcConvertible
		if errors.As(err, &conv) {
			return nil, conv.ToGRPC()
		}

		return nil, o.Fallback(err)
	}
}

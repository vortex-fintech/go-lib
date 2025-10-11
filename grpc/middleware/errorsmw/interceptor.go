package errorsmw

import (
	"context"
	"errors"

	gliberrors "github.com/vortex-fintech/go-lib/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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
		Fallback: func(_ error) error { return gliberrors.Internal().ToGRPC() },
	}
	for _, f := range opts {
		f(&o)
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if err == nil {
			return resp, nil
		}
		return nil, toGRPC(err, o.Fallback)
	}
}

func Stream(opts ...Option) grpc.StreamServerInterceptor {
	o := Options{
		Fallback: func(_ error) error { return gliberrors.Internal().ToGRPC() },
	}
	for _, f := range opts {
		f(&o)
	}
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err := handler(srv, ss); err != nil {
			return toGRPC(err, o.Fallback)
		}
		return nil
	}
}

func toGRPC(err error, fallback func(error) error) error {
	// Уже gRPC status?
	if _, ok := status.FromError(err); ok {
		return err
	}
	// Наш тип с ToGRPC()
	var conv grpcConvertible
	if errors.As(err, &conv) {
		return conv.ToGRPC()
	}
	// Один доменный
	if gliberrors.IsDomainError(err) {
		return gliberrors.ConvertDomainToValidation(err).ToGRPC()
	}
	// Батч доменных
	if de, ok := err.(gliberrors.DomainErrors); ok {
		return gliberrors.ConvertDomainErrorsToValidation(de).ToGRPC()
	}
	// Контекст
	if errors.Is(err, context.Canceled) {
		return status.Error(codes.Canceled, "Request canceled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return status.Error(codes.DeadlineExceeded, "Deadline exceeded")
	}
	// Фоллбек
	return fallback(err)
}

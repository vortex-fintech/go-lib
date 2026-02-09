package recoverymw

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Options struct {
	OnPanic func(ctx context.Context, method string, recovered any)
}

func Unary(opts Options) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ any, err error) {
		defer func() {
			r := recover()
			if r == nil {
				return
			}
			if opts.OnPanic != nil {
				opts.OnPanic(ctx, info.FullMethod, r)
			}
			err = status.Error(codes.Internal, "internal server error")
		}()
		return handler(ctx, req)
	}
}

func PanicString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case error:
		return t.Error()
	default:
		return fmt.Sprintf("%v", t)
	}
}

package contextcancel

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {

		if err := ctx.Err(); err != nil {
			return nil, status.FromContextError(err).Err()
		}

		resp, err := handler(ctx, req)

		if err == nil {
			if cerr := ctx.Err(); cerr != nil {
				return nil, status.FromContextError(cerr).Err()
			}
		}

		return resp, err
	}
}

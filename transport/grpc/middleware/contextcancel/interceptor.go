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

func Stream() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {

		if err := ss.Context().Err(); err != nil {
			return status.FromContextError(err).Err()
		}

		err := handler(srv, ss)

		if err == nil {
			if cerr := ss.Context().Err(); cerr != nil {
				return status.FromContextError(cerr).Err()
			}
		}

		return err
	}
}

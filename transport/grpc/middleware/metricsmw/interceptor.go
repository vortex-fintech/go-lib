package metrics

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FullReporter interface {
	ObserveRPCFull(ctx context.Context, fullMethod string, code codes.Code, secs float64)
}

func UnaryFull(r FullReporter) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)

		code := status.Code(err)
		if err == nil {
			code = codes.OK
		}
		r.ObserveRPCFull(ctx, info.FullMethod, code, time.Since(start).Seconds())
		return resp, err
	}
}

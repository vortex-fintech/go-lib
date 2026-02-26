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
	if r == nil {
		return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			return handler(ctx, req)
		}
	}
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

func StreamFull(r FullReporter) grpc.StreamServerInterceptor {
	if r == nil {
		return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			return handler(srv, ss)
		}
	}
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()
		err := handler(srv, ss)

		code := status.Code(err)
		if err == nil {
			code = codes.OK
		}
		r.ObserveRPCFull(ss.Context(), info.FullMethod, code, time.Since(start).Seconds())
		return err
	}
}

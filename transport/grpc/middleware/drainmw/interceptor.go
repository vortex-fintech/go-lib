package drainmw

import (
	"context"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Controller struct {
	draining atomic.Bool
}

func NewController() *Controller {
	return &Controller{}
}

func (c *Controller) StartDraining() {
	if c == nil {
		return
	}
	c.draining.Store(true)
}

func (c *Controller) IsDraining() bool {
	if c == nil {
		return false
	}
	return c.draining.Load()
}

func Unary(c *Controller, isMutating func(fullMethod string) bool) grpc.UnaryServerInterceptor {
	if isMutating == nil {
		isMutating = func(string) bool { return true }
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if c != nil && c.IsDraining() && isMutating(info.FullMethod) {
			return nil, status.Error(codes.Unavailable, "server is draining, retry later")
		}
		return handler(ctx, req)
	}
}

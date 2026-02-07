package contextcancel

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func call(t *testing.T, ctx context.Context, itc grpc.UnaryServerInterceptor, h grpc.UnaryHandler) (any, error) {
	t.Helper()
	return itc(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, h)
}

func TestUnary_CancelledBeforeHandler(t *testing.T) {
	itc := Unary()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := call(t, ctx, itc, func(ctx context.Context, req any) (any, error) {
		t.Fatal("handler should not be called")
		return nil, nil
	})
	if status.Code(err) != codes.Canceled {
		t.Fatalf("want Canceled, got %v", err)
	}
}

func TestUnary_CancelledAfterSuccess(t *testing.T) {
	itc := Unary()
	ctx, cancel := context.WithCancel(context.Background())

	resp, err := call(t, ctx, itc, func(ctx context.Context, req any) (any, error) {
		go func() {
			time.Sleep(1 * time.Millisecond)
			cancel()
		}()
		time.Sleep(2 * time.Millisecond)
		return "ok", nil
	})

	if status.Code(err) != codes.Canceled || resp != nil {
		t.Fatalf("want Canceled after handler, got resp=%v err=%v", resp, err)
	}
}

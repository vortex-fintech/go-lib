package deadlinemw

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
)

func TestUnary_SetsDefaultWhenNoDeadline(t *testing.T) {
	i := Unary(Config{DefaultTimeout: 50 * time.Millisecond})
	_, err := i(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/m"}, func(ctx context.Context, req any) (any, error) {
		dl, ok := ctx.Deadline()
		if !ok {
			t.Fatalf("expected deadline")
		}
		if time.Until(dl) <= 0 {
			t.Fatalf("deadline already expired")
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnary_RespectsShorterExistingDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	i := Unary(Config{DefaultTimeout: 80 * time.Millisecond, MaxTimeout: 100 * time.Millisecond})
	_, err := i(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/m"}, func(ctx context.Context, req any) (any, error) {
		dl, ok := ctx.Deadline()
		if !ok {
			t.Fatalf("expected deadline")
		}
		if time.Until(dl) > 40*time.Millisecond {
			t.Fatalf("expected to keep short existing deadline")
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnary_AppliesMaxWhenNoDeadlineAndNoDefault(t *testing.T) {
	i := Unary(Config{MaxTimeout: 30 * time.Millisecond})
	_, err := i(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/m"}, func(ctx context.Context, req any) (any, error) {
		dl, ok := ctx.Deadline()
		if !ok {
			t.Fatalf("expected deadline")
		}
		if time.Until(dl) <= 0 {
			t.Fatalf("deadline already expired")
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

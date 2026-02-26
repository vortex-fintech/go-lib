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

func TestUnary_MaxTimeoutCapsDefaultTimeout(t *testing.T) {
	i := Unary(Config{DefaultTimeout: 5 * time.Second, MaxTimeout: 50 * time.Millisecond})
	_, err := i(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/m"}, func(ctx context.Context, req any) (any, error) {
		dl, ok := ctx.Deadline()
		if !ok {
			t.Fatalf("expected deadline")
		}
		remaining := time.Until(dl)
		if remaining > 100*time.Millisecond {
			t.Fatalf("expected deadline capped to MaxTimeout, got %v", remaining)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnary_MethodTimeout(t *testing.T) {
	i := Unary(Config{
		DefaultTimeout: 5 * time.Second,
		MethodTimeouts: map[string]time.Duration{
			"/svc/slow": 10 * time.Second,
			"/svc/fast": 50 * time.Millisecond,
		},
	})

	_, err := i(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/fast"}, func(ctx context.Context, req any) (any, error) {
		dl, ok := ctx.Deadline()
		if !ok {
			t.Fatalf("expected deadline")
		}
		remaining := time.Until(dl)
		if remaining > 100*time.Millisecond {
			t.Fatalf("expected method-specific deadline, got %v", remaining)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnary_MaxTimeoutCapsClientDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	i := Unary(Config{MaxTimeout: 50 * time.Millisecond})
	_, err := i(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/m"}, func(ctx context.Context, req any) (any, error) {
		dl, ok := ctx.Deadline()
		if !ok {
			t.Fatalf("expected deadline")
		}
		remaining := time.Until(dl)
		if remaining > 100*time.Millisecond {
			t.Fatalf("expected client deadline capped to MaxTimeout, got %v", remaining)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnary_NoDeadlineWhenNoConfig(t *testing.T) {
	i := Unary(Config{})
	_, err := i(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/m"}, func(ctx context.Context, req any) (any, error) {
		_, ok := ctx.Deadline()
		if ok {
			t.Fatalf("expected no deadline when config is empty")
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

package contextcancel

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func callUnary(t *testing.T, ctx context.Context, h grpc.UnaryHandler) (any, error) {
	t.Helper()
	return Unary()(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, h)
}

func TestUnary_CancelledBeforeHandler(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := callUnary(t, ctx, func(ctx context.Context, req any) (any, error) {
		t.Fatal("handler should not be called")
		return nil, nil
	})
	if status.Code(err) != codes.Canceled {
		t.Fatalf("want Canceled, got %v", err)
	}
}

func TestUnary_CancelledAfterSuccess(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		<-done
		cancel()
	}()

	resp, err := callUnary(t, ctx, func(ctx context.Context, req any) (any, error) {
		close(done)
		<-ctx.Done()
		return "ok", nil
	})

	if status.Code(err) != codes.Canceled {
		t.Fatalf("want Canceled, got %v", err)
	}
	if resp != nil {
		t.Fatalf("expected nil response, got %v", resp)
	}
}

func TestUnary_NotCancelled(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	resp, err := callUnary(t, ctx, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "ok" {
		t.Fatalf("expected ok, got %v", resp)
	}
}

func TestUnary_HandlerErrorNotOverwritten(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := callUnary(t, ctx, func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.Internal, "boom")
	})
	if status.Code(err) != codes.Canceled {
		t.Fatalf("want Canceled (context check before handler), got %v", err)
	}
}

type mockStream struct {
	ctx context.Context
}

func (m *mockStream) Context() context.Context        { return m.ctx }
func (m *mockStream) SetHeader(md metadata.MD) error  { return nil }
func (m *mockStream) SendHeader(md metadata.MD) error { return nil }
func (m *mockStream) SetTrailer(md metadata.MD)       {}
func (m *mockStream) SendMsg(m2 any) error            { return nil }
func (m *mockStream) RecvMsg(m2 any) error            { return nil }

func callStream(t *testing.T, ctx context.Context, h grpc.StreamHandler) error {
	t.Helper()
	ss := &mockStream{ctx: ctx}
	return Stream()(nil, ss, &grpc.StreamServerInfo{FullMethod: "/svc/Stream"}, h)
}

func TestStream_CancelledBeforeHandler(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := callStream(t, ctx, func(srv any, ss grpc.ServerStream) error {
		t.Fatal("handler should not be called")
		return nil
	})
	if status.Code(err) != codes.Canceled {
		t.Fatalf("want Canceled, got %v", err)
	}
}

func TestStream_NotCancelled(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	err := callStream(t, ctx, func(srv any, ss grpc.ServerStream) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStream_HandlerError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	err := callStream(t, ctx, func(srv any, ss grpc.ServerStream) error {
		return status.Error(codes.Internal, "boom")
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("want Internal, got %v", err)
	}
}

func TestStream_CancelledAfterSuccess(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		<-done
		cancel()
	}()

	err := callStream(t, ctx, func(srv any, ss grpc.ServerStream) error {
		close(done)
		<-ctx.Done()
		return nil
	})

	if status.Code(err) != codes.Canceled {
		t.Fatalf("want Canceled, got %v", err)
	}
}

func TestUnary_DeadlineExceeded(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	_, err := callUnary(t, ctx, func(ctx context.Context, req any) (any, error) {
		t.Fatal("handler should not be called")
		return nil, nil
	})
	if status.Code(err) != codes.DeadlineExceeded {
		t.Fatalf("want DeadlineExceeded, got %v", err)
	}
}

func TestStream_DeadlineExceeded(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	err := callStream(t, ctx, func(srv any, ss grpc.ServerStream) error {
		t.Fatal("handler should not be called")
		return nil
	})
	if status.Code(err) != codes.DeadlineExceeded {
		t.Fatalf("want DeadlineExceeded, got %v", err)
	}
}

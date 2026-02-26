package recoverymw

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type mockStream struct {
	ctx context.Context
}

func (m *mockStream) Context() context.Context        { return m.ctx }
func (m *mockStream) SetHeader(md metadata.MD) error  { return nil }
func (m *mockStream) SendHeader(md metadata.MD) error { return nil }
func (m *mockStream) SetTrailer(md metadata.MD)       {}
func (m *mockStream) SendMsg(m2 any) error            { return nil }
func (m *mockStream) RecvMsg(m2 any) error            { return nil }

func TestUnary_RecoversPanic(t *testing.T) {
	called := false
	i := Unary(Options{OnPanic: func(context.Context, string, any) { called = true }})
	_, err := i(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/x/y/z"}, func(context.Context, any) (any, error) {
		panic("boom")
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal, got %v", status.Code(err))
	}
	if !called {
		t.Fatalf("expected OnPanic to be called")
	}
}

func TestPanicString(t *testing.T) {
	t.Parallel()

	if got := PanicString("x"); got != "x" {
		t.Fatalf("unexpected string panic text: %q", got)
	}
	if got := PanicString(errors.New("boom")); got != "boom" {
		t.Fatalf("unexpected error panic text: %q", got)
	}
	if got := PanicString(42); got != "42" {
		t.Fatalf("unexpected fallback panic text: %q", got)
	}
}

func TestUnary_NilOnPanic(t *testing.T) {
	i := Unary(Options{})
	_, err := i(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/x/y/z"}, func(context.Context, any) (any, error) {
		panic("boom")
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal, got %v", status.Code(err))
	}
}

func TestUnary_NoPanic(t *testing.T) {
	i := Unary(Options{})
	_, err := i(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/x/y/z"}, func(context.Context, any) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStream_RecoversPanic(t *testing.T) {
	called := false
	i := Stream(Options{OnPanic: func(context.Context, string, any) { called = true }})
	ss := &mockStream{ctx: context.Background()}
	err := i("srv", ss, &grpc.StreamServerInfo{FullMethod: "/x/y/z"}, func(any, grpc.ServerStream) error {
		panic("boom")
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal, got %v", status.Code(err))
	}
	if !called {
		t.Fatalf("expected OnPanic to be called")
	}
}

func TestStream_NilOnPanic(t *testing.T) {
	i := Stream(Options{})
	ss := &mockStream{ctx: context.Background()}
	err := i("srv", ss, &grpc.StreamServerInfo{FullMethod: "/x/y/z"}, func(any, grpc.ServerStream) error {
		panic("boom")
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal, got %v", status.Code(err))
	}
}

func TestStream_NoPanic(t *testing.T) {
	i := Stream(Options{})
	ss := &mockStream{ctx: context.Background()}
	err := i("srv", ss, &grpc.StreamServerInfo{FullMethod: "/x/y/z"}, func(any, grpc.ServerStream) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

package recoverymw

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

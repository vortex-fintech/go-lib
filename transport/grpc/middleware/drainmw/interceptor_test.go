package drainmw

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnary_BlocksMutatingWhenDraining(t *testing.T) {
	c := NewController()
	c.StartDraining()
	i := Unary(c, func(string) bool { return true })
	_, err := i(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/M"}, func(context.Context, any) (any, error) {
		return nil, nil
	})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("expected Unavailable, got %v", status.Code(err))
	}
}

func TestUnary_AllowsReadWhenDraining(t *testing.T) {
	c := NewController()
	c.StartDraining()
	i := Unary(c, func(string) bool { return false })
	_, err := i(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/R"}, func(context.Context, any) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnary_NilControllerAllows(t *testing.T) {
	i := Unary(nil, nil)
	_, err := i(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/M"}, func(context.Context, any) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

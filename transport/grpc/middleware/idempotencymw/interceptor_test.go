package idempotencymw

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestUnary_PutsMetadataIntoContext(t *testing.T) {
	i := Unary(Config{
		IsMethodEnabled: func(string) bool { return true },
		ResolvePrincipal: func(context.Context, metadata.MD) string {
			return "principal-1"
		},
	})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("idempotency-key", "k-1"))

	_, err := i(ctx, &emptypb.Empty{}, &grpc.UnaryServerInfo{FullMethod: "/svc/method"}, func(ctx context.Context, req any) (any, error) {
		m, ok := FromContext(ctx)
		if !ok {
			t.Fatalf("expected metadata in context")
		}
		if m.IdempotencyKey != "k-1" || m.GRPCMethod != "/svc/method" || m.Principal != "principal-1" || m.RequestHash == "" {
			t.Fatalf("unexpected metadata %+v", m)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnary_RequireKey(t *testing.T) {
	i := Unary(Config{RequireKey: true, IsMethodEnabled: func(string) bool { return true }})
	_, err := i(context.Background(), &emptypb.Empty{}, &grpc.UnaryServerInfo{FullMethod: "/svc/method"}, func(context.Context, any) (any, error) {
		return nil, nil
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestUnary_DisabledMethodSkips(t *testing.T) {
	i := Unary(Config{IsMethodEnabled: func(string) bool { return false }, RequireKey: true})
	_, err := i(context.Background(), &emptypb.Empty{}, &grpc.UnaryServerInfo{FullMethod: "/svc/method"}, func(context.Context, any) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnary_TooLongKey(t *testing.T) {
	i := Unary(Config{MaxKeyLength: 3, IsMethodEnabled: func(string) bool { return true }})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("idempotency-key", "1234"))
	_, err := i(ctx, &emptypb.Empty{}, &grpc.UnaryServerInfo{FullMethod: "/svc/method"}, func(context.Context, any) (any, error) {
		return nil, nil
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestFromContext_Missing(t *testing.T) {
	if _, ok := FromContext(context.Background()); ok {
		t.Fatalf("expected missing metadata in empty context")
	}
}

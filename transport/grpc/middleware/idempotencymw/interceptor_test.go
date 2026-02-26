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

func TestUnary_CustomHeader(t *testing.T) {
	i := Unary(Config{
		Header:          "x-idempotency-key",
		IsMethodEnabled: func(string) bool { return true },
	})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-idempotency-key", "k-1"))

	_, err := i(ctx, &emptypb.Empty{}, &grpc.UnaryServerInfo{FullMethod: "/svc/method"}, func(ctx context.Context, req any) (any, error) {
		m, ok := FromContext(ctx)
		if !ok {
			t.Fatalf("expected metadata in context")
		}
		if m.IdempotencyKey != "k-1" {
			t.Fatalf("expected key k-1, got %s", m.IdempotencyKey)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnary_NonProtobufRequest(t *testing.T) {
	i := Unary(Config{IsMethodEnabled: func(string) bool { return true }})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("idempotency-key", "k-1"))

	_, err := i(ctx, "not a protobuf", &grpc.UnaryServerInfo{FullMethod: "/svc/method"}, func(context.Context, any) (any, error) {
		return nil, nil
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal for non-protobuf, got %v", status.Code(err))
	}
}

func TestUnary_SkipsWhenNoKey(t *testing.T) {
	i := Unary(Config{IsMethodEnabled: func(string) bool { return true }})

	_, err := i(context.Background(), &emptypb.Empty{}, &grpc.UnaryServerInfo{FullMethod: "/svc/method"}, func(ctx context.Context, req any) (any, error) {
		if _, ok := FromContext(ctx); ok {
			t.Fatalf("expected no metadata when no key provided")
		}
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnary_SameRequestSameHash(t *testing.T) {
	i := Unary(Config{IsMethodEnabled: func(string) bool { return true }})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("idempotency-key", "k-1"))

	var hash1, hash2 string

	_, _ = i(ctx, &emptypb.Empty{}, &grpc.UnaryServerInfo{FullMethod: "/svc/method"}, func(ctx context.Context, req any) (any, error) {
		m, _ := FromContext(ctx)
		hash1 = m.RequestHash
		return nil, nil
	})

	_, _ = i(ctx, &emptypb.Empty{}, &grpc.UnaryServerInfo{FullMethod: "/svc/method"}, func(ctx context.Context, req any) (any, error) {
		m, _ := FromContext(ctx)
		hash2 = m.RequestHash
		return nil, nil
	})

	if hash1 != hash2 {
		t.Fatalf("same request should produce same hash: %s != %s", hash1, hash2)
	}
}

func TestUnary_Defaults(t *testing.T) {
	i := Unary(Config{})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("idempotency-key", "k-1"))
	_, err := i(ctx, &emptypb.Empty{}, &grpc.UnaryServerInfo{FullMethod: "/svc/method"}, func(ctx context.Context, req any) (any, error) {
		m, ok := FromContext(ctx)
		if !ok {
			t.Fatalf("expected metadata")
		}
		if m.Principal != "unknown" {
			t.Fatalf("expected default principal 'unknown', got %s", m.Principal)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

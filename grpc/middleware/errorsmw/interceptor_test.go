package errorsmw

import (
	"context"
	"errors"
	"testing"

	gliberrors "github.com/vortex-fintech/go-lib/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeConv struct{}

func (fakeConv) Error() string { return "fake" }
func (fakeConv) ToGRPC() error { return status.Error(codes.InvalidArgument, "bad input") }

func call(t *testing.T, itc grpc.UnaryServerInterceptor, h grpc.UnaryHandler) (any, error) {
	t.Helper()
	return itc(nil, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, h)
}

func TestUnary_PassesStatusAsIs(t *testing.T) {
	itc := Unary()
	_, err := call(t, itc, func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.NotFound, "nope")
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("want NotFound, got %v", err)
	}
}

func TestUnary_UsesToGRPC(t *testing.T) {
	itc := Unary()
	_, err := call(t, itc, func(ctx context.Context, req any) (any, error) {
		return nil, fakeConv{}
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("want InvalidArgument, got %v", err)
	}
}

func TestUnary_FallbackInternal(t *testing.T) {
	itc := Unary()
	_, err := call(t, itc, func(ctx context.Context, req any) (any, error) {
		return nil, errors.New("boom")
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("want Internal, got %v", err)
	}
}

func TestUnary_CustomFallback(t *testing.T) {
	itc := Unary(WithFallback(func(err error) error {
		return status.Error(codes.ResourceExhausted, "throttled")
	}))
	_, err := call(t, itc, func(ctx context.Context, req any) (any, error) {
		return nil, errors.New("boom")
	})
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("want ResourceExhausted, got %v", err)
	}
}

func TestToGRPC_DomainErrorsBatch(t *testing.T) {
	de := gliberrors.DomainErrors{
		{Field: "email", Reason: "invalid_email"},
		{Field: "password", Reason: "too_short"},
	}
	out := toGRPC(de, func(error) error { return nil })
	st, _ := status.FromError(out)
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("domain batch should map to InvalidArgument")
	}
}

func TestToGRPC_PassthroughAndContext(t *testing.T) {
	out := toGRPC(context.Canceled, func(error) error { return nil })
	st, _ := status.FromError(out)
	if st.Code() != codes.Canceled {
		t.Fatalf("context.Canceled → Canceled expected")
	}
}

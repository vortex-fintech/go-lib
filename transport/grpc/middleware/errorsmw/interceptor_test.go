package errorsmw

import (
	"context"
	"errors"
	"testing"

	gliberrors "github.com/vortex-fintech/go-lib/foundation/errors"
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
	de := gliberrors.ValidationFields(map[string]string{
		"email":    "invalid_email",
		"password": "too_short",
	})
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

func TestToGRPC_DeadlineExceeded(t *testing.T) {
	out := toGRPC(context.DeadlineExceeded, func(error) error { return nil })
	st, _ := status.FromError(out)
	if st.Code() != codes.DeadlineExceeded {
		t.Fatalf("context.DeadlineExceeded → DeadlineExceeded expected")
	}
}

func TestStream_PassesStatusAsIs(t *testing.T) {
	itc := Stream()
	err := itc(nil, nil, &grpc.StreamServerInfo{FullMethod: "/svc/Method"}, func(any, grpc.ServerStream) error {
		return status.Error(codes.NotFound, "nope")
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("want NotFound, got %v", err)
	}
}

func TestStream_UsesToGRPC(t *testing.T) {
	itc := Stream()
	err := itc(nil, nil, &grpc.StreamServerInfo{FullMethod: "/svc/Method"}, func(any, grpc.ServerStream) error {
		return fakeConv{}
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("want InvalidArgument, got %v", err)
	}
}

func TestStream_FallbackInternal(t *testing.T) {
	itc := Stream()
	err := itc(nil, nil, &grpc.StreamServerInfo{FullMethod: "/svc/Method"}, func(any, grpc.ServerStream) error {
		return errors.New("boom")
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("want Internal, got %v", err)
	}
}

func TestStream_CustomFallback(t *testing.T) {
	itc := Stream(WithFallback(func(err error) error {
		return status.Error(codes.ResourceExhausted, "throttled")
	}))
	err := itc(nil, nil, &grpc.StreamServerInfo{FullMethod: "/svc/Method"}, func(any, grpc.ServerStream) error {
		return errors.New("boom")
	})
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("want ResourceExhausted, got %v", err)
	}
}

func TestStream_ContextCanceled(t *testing.T) {
	itc := Stream()
	err := itc(nil, nil, &grpc.StreamServerInfo{FullMethod: "/svc/Method"}, func(any, grpc.ServerStream) error {
		return context.Canceled
	})
	if status.Code(err) != codes.Canceled {
		t.Fatalf("want Canceled, got %v", err)
	}
}

func TestUnary_ErrorResponseDirect(t *testing.T) {
	itc := Unary()
	_, err := call(t, itc, func(ctx context.Context, req any) (any, error) {
		return nil, gliberrors.NotFound()
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("want NotFound, got %v", err)
	}
}

func TestUnary_InvariantError(t *testing.T) {
	itc := Unary()
	_, err := call(t, itc, func(ctx context.Context, req any) (any, error) {
		return nil, gliberrors.DomainInvariant("email", "invalid_format")
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("want InvalidArgument for DomainInvariant, got %v", err)
	}
}

func TestUnary_StateInvariant(t *testing.T) {
	itc := Unary()
	_, err := call(t, itc, func(ctx context.Context, req any) (any, error) {
		return nil, gliberrors.StateInvariant(nil, "status", "invalid_transition")
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("want FailedPrecondition for StateInvariant, got %v", err)
	}
}

package metrics

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeFullReporter struct {
	calls int
	ctx   context.Context
	full  string
	code  codes.Code
	secs  float64
}

func (f *fakeFullReporter) ObserveRPCFull(ctx context.Context, fullMethod string, code codes.Code, secs float64) {
	f.calls++
	f.ctx = ctx
	f.full = fullMethod
	f.code = code
	f.secs = secs
}

func TestUnaryFull_OK(t *testing.T) {
	t.Parallel()

	frep := &fakeFullReporter{}
	intc := UnaryFull(frep)

	info := &grpc.UnaryServerInfo{FullMethod: "/pkg.Service/Method"}
	handler := func(ctx context.Context, req any) (any, error) {
		time.Sleep(5 * time.Millisecond)
		return "resp", nil
	}

	resp, err := intc(context.Background(), "req", info, handler)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp != "resp" {
		t.Fatalf("unexpected resp: %v", resp)
	}

	if frep.calls != 1 {
		t.Fatalf("reporter not called, calls=%d", frep.calls)
	}
	if frep.full != "/pkg.Service/Method" {
		t.Fatalf("full method mismatch: %q", frep.full)
	}
	if frep.code != codes.OK {
		t.Fatalf("code mismatch: got %v want %v", frep.code, codes.OK)
	}
	if frep.secs < 0 {
		t.Fatalf("duration must be >= 0, got %f", frep.secs)
	}
}

func TestUnaryFull_Error(t *testing.T) {
	t.Parallel()

	frep := &fakeFullReporter{}
	intc := UnaryFull(frep)

	info := &grpc.UnaryServerInfo{FullMethod: "/pkg.Service/Boom"}
	wantErr := status.Error(codes.Internal, "boom")
	handler := func(ctx context.Context, req any) (any, error) {
		return nil, wantErr
	}

	resp, err := intc(context.Background(), "req", info, handler)
	if status.Code(err) != codes.Internal {
		t.Fatalf("code mismatch: got %v want %v", status.Code(err), codes.Internal)
	}
	if resp != nil {
		t.Fatalf("unexpected resp: %v", resp)
	}

	if frep.calls != 1 {
		t.Fatalf("reporter not called, calls=%d", frep.calls)
	}
	if frep.full != "/pkg.Service/Boom" {
		t.Fatalf("full method mismatch: %q", frep.full)
	}
	if frep.code != codes.Internal {
		t.Fatalf("code mismatch: got %v want %v", frep.code, codes.Internal)
	}
}

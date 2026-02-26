package metrics

import (
	"context"
	"testing"
	"time"

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

func TestStreamFull_OK(t *testing.T) {
	t.Parallel()

	frep := &fakeFullReporter{}
	intc := StreamFull(frep)

	info := &grpc.StreamServerInfo{FullMethod: "/pkg.Service/Stream"}
	ss := &mockStream{ctx: context.Background()}
	handler := func(srv any, stream grpc.ServerStream) error {
		return nil
	}

	err := intc("srv", ss, info, handler)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if frep.calls != 1 {
		t.Fatalf("reporter not called, calls=%d", frep.calls)
	}
	if frep.full != "/pkg.Service/Stream" {
		t.Fatalf("full method mismatch: %q", frep.full)
	}
	if frep.code != codes.OK {
		t.Fatalf("code mismatch: got %v want %v", frep.code, codes.OK)
	}
	if frep.secs < 0 {
		t.Fatalf("duration must be >= 0, got %f", frep.secs)
	}
}

func TestStreamFull_Error(t *testing.T) {
	t.Parallel()

	frep := &fakeFullReporter{}
	intc := StreamFull(frep)

	info := &grpc.StreamServerInfo{FullMethod: "/pkg.Service/Boom"}
	ss := &mockStream{ctx: context.Background()}
	wantErr := status.Error(codes.Internal, "boom")
	handler := func(srv any, stream grpc.ServerStream) error {
		return wantErr
	}

	err := intc("srv", ss, info, handler)
	if status.Code(err) != codes.Internal {
		t.Fatalf("code mismatch: got %v want %v", status.Code(err), codes.Internal)
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

func TestUnaryFull_NilReporter(t *testing.T) {
	t.Parallel()

	intc := UnaryFull(nil)

	resp, err := intc(context.Background(), "req", &grpc.UnaryServerInfo{FullMethod: "/pkg.Service/Method"}, func(ctx context.Context, req any) (any, error) {
		return "resp", nil
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp != "resp" {
		t.Fatalf("unexpected resp: %v", resp)
	}
}

func TestStreamFull_NilReporter(t *testing.T) {
	t.Parallel()

	intc := StreamFull(nil)
	err := intc("srv", &mockStream{ctx: context.Background()}, &grpc.StreamServerInfo{FullMethod: "/pkg.Service/Stream"}, func(srv any, stream grpc.ServerStream) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

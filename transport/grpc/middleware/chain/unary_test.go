package chain_test

import (
	"context"
	"testing"

	"github.com/vortex-fintech/go-lib/transport/grpc/middleware/chain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type callRecorder struct {
	calls []string
}

func (r *callRecorder) unary(name string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		r.calls = append(r.calls, name)
		return handler(ctx, req)
	}
}

func (r *callRecorder) stream(name string) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		r.calls = append(r.calls, name)
		return handler(srv, ss)
	}
}

func TestDefault_ReturnsServerOption(t *testing.T) {
	t.Parallel()

	opt := chain.Default(chain.Options{})
	if opt == nil {
		t.Fatal("expected non-nil ServerOption")
	}
}

func TestDefault_EmptyOptions(t *testing.T) {
	t.Parallel()

	opt := chain.Options{}
	_ = chain.Default(opt)
}

func TestDefault_WithPreAndPost(t *testing.T) {
	t.Parallel()

	rec := &callRecorder{}
	opt := chain.Options{
		Pre:  []grpc.UnaryServerInterceptor{rec.unary("pre1"), rec.unary("pre2")},
		Post: []grpc.UnaryServerInterceptor{rec.unary("post1")},
	}

	_ = chain.Default(opt)
}

func TestDefault_WithAuthz(t *testing.T) {
	t.Parallel()

	rec := &callRecorder{}
	opt := chain.Options{
		Pre:              []grpc.UnaryServerInterceptor{rec.unary("pre")},
		AuthzInterceptor: rec.unary("authz"),
		Post:             []grpc.UnaryServerInterceptor{rec.unary("post")},
	}

	_ = chain.Default(opt)
}

func TestDefault_WithCircuitBreaker(t *testing.T) {
	t.Parallel()

	opt := chain.Options{
		CircuitBreaker: nil,
	}

	_ = chain.Default(opt)
}

func TestDefault_DisableCtxCancel(t *testing.T) {
	t.Parallel()

	opt := chain.Options{
		DisableCtxCancel: true,
	}

	_ = chain.Default(opt)
}

func TestDefault_DisableErrors(t *testing.T) {
	t.Parallel()

	opt := chain.Options{
		DisableErrors: true,
	}

	_ = chain.Default(opt)
}

func TestDefault_FullOptions(t *testing.T) {
	t.Parallel()

	rec := &callRecorder{}
	opt := chain.Options{
		Pre:              []grpc.UnaryServerInterceptor{rec.unary("pre")},
		AuthzInterceptor: rec.unary("authz"),
		Post:             []grpc.UnaryServerInterceptor{rec.unary("post")},
	}

	_ = chain.Default(opt)
}

func TestDefaultStream_ReturnsServerOption(t *testing.T) {
	t.Parallel()

	opt := chain.DefaultStream(chain.StreamOptions{})
	if opt == nil {
		t.Fatal("expected non-nil ServerOption")
	}
}

func TestDefaultStream_EmptyOptions(t *testing.T) {
	t.Parallel()

	opt := chain.StreamOptions{}
	_ = chain.DefaultStream(opt)
}

func TestDefaultStream_WithPreAndPost(t *testing.T) {
	t.Parallel()

	rec := &callRecorder{}
	opt := chain.StreamOptions{
		Pre:  []grpc.StreamServerInterceptor{rec.stream("pre1"), rec.stream("pre2")},
		Post: []grpc.StreamServerInterceptor{rec.stream("post1")},
	}

	_ = chain.DefaultStream(opt)
}

func TestDefaultStream_WithAuthz(t *testing.T) {
	t.Parallel()

	rec := &callRecorder{}
	opt := chain.StreamOptions{
		Pre:              []grpc.StreamServerInterceptor{rec.stream("pre")},
		AuthzInterceptor: rec.stream("authz"),
		Post:             []grpc.StreamServerInterceptor{rec.stream("post")},
	}

	_ = chain.DefaultStream(opt)
}

func TestDefaultStream_DisableCtxCancel(t *testing.T) {
	t.Parallel()

	opt := chain.StreamOptions{
		DisableCtxCancel: true,
	}

	_ = chain.DefaultStream(opt)
}

func TestDefaultStream_DisableErrors(t *testing.T) {
	t.Parallel()

	opt := chain.StreamOptions{
		DisableErrors: true,
	}

	_ = chain.DefaultStream(opt)
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

func TestMockStream_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ grpc.ServerStream = &mockStream{ctx: context.Background()}
}

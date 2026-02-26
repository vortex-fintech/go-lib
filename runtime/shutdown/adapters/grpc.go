package adapters

import (
	"context"
	"errors"
	"net"

	"google.golang.org/grpc"
)

// GRPC adapts *grpc.Server to the shutdown.Server interface.
type GRPC struct {
	Srv     *grpc.Server
	Lis     net.Listener
	NameStr string
}

// Name returns the server name. Returns "grpc" if NameStr is empty.
func (g *GRPC) Name() string {
	if g.NameStr == "" {
		return "grpc"
	}
	return g.NameStr
}

// Serve starts the gRPC server and blocks until ctx is cancelled or an error occurs.
// Returns an error if Srv or Lis is nil.
func (g *GRPC) Serve(ctx context.Context) error {
	if g.Srv == nil {
		return errors.New("grpc adapter: Srv is nil")
	}
	if g.Lis == nil {
		return errors.New("grpc adapter: Lis is nil")
	}

	errCh := make(chan error, 1)
	go func() { errCh <- g.Srv.Serve(g.Lis) }()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

// GracefulStopWithTimeout gracefully shuts down the server.
// Returns an error if Srv is nil.
func (g *GRPC) GracefulStopWithTimeout(ctx context.Context) error {
	if g.Srv == nil {
		return errors.New("grpc adapter: Srv is nil")
	}

	done := make(chan struct{}, 1)
	go func() {
		g.Srv.GracefulStop()
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

// ForceStop immediately stops the server.
// No-op if Srv is nil.
func (g *GRPC) ForceStop() {
	if g.Srv != nil {
		g.Srv.Stop()
	}
}

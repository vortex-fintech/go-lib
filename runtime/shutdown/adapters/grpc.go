package adapters

import (
	"context"
	"net"

	"google.golang.org/grpc"
)

type GRPC struct {
	Srv     *grpc.Server
	Lis     net.Listener
	NameStr string
}

func (g *GRPC) Name() string {
	if g.NameStr == "" {
		return "grpc"
	}
	return g.NameStr
}

func (g *GRPC) Serve(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() { errCh <- g.Srv.Serve(g.Lis) }()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (g *GRPC) GracefulStopWithTimeout(ctx context.Context) error {
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

func (g *GRPC) ForceStop() {
	g.Srv.Stop()
}

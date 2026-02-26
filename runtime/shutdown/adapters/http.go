package adapters

import (
	"context"
	"errors"
	"net"
	"net/http"
)

// HTTP adapts *http.Server to the shutdown.Server interface.
// If Lis is nil, Serve uses ListenAndServe(); otherwise it uses Serve(Lis).
type HTTP struct {
	Srv     *http.Server
	Lis     net.Listener
	NameStr string
}

// Name returns the server name. Returns "http" if NameStr is empty.
func (h *HTTP) Name() string {
	if h.NameStr == "" {
		return "http"
	}
	return h.NameStr
}

// Serve starts the HTTP server and blocks until ctx is cancelled or an error occurs.
// Returns an error if Srv is nil.
func (h *HTTP) Serve(ctx context.Context) error {
	if h.Srv == nil {
		return errors.New("http adapter: Srv is nil")
	}

	errCh := make(chan error, 1)

	h.Srv.BaseContext = func(_ net.Listener) context.Context { return ctx }

	go func() {
		if h.Lis != nil {
			errCh <- h.Srv.Serve(h.Lis)
			return
		}
		errCh <- h.Srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

// GracefulStopWithTimeout gracefully shuts down the server.
// Returns an error if Srv is nil.
func (h *HTTP) GracefulStopWithTimeout(ctx context.Context) error {
	if h.Srv == nil {
		return errors.New("http adapter: Srv is nil")
	}
	return h.Srv.Shutdown(ctx)
}

// ForceStop immediately closes the server.
// No-op if Srv is nil.
func (h *HTTP) ForceStop() {
	if h.Srv != nil {
		_ = h.Srv.Close()
	}
}

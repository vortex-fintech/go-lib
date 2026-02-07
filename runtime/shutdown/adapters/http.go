package adapters

import (
	"context"
	"net"
	"net/http"
)

type HTTP struct {
	Srv     *http.Server
	Lis     net.Listener
	NameStr string
}

func (h *HTTP) Name() string {
	if h.NameStr == "" {
		return "http"
	}
	return h.NameStr
}

func (h *HTTP) Serve(ctx context.Context) error {
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

func (h *HTTP) GracefulStopWithTimeout(ctx context.Context) error {
	return h.Srv.Shutdown(ctx)
}

func (h *HTTP) ForceStop() {
	_ = h.Srv.Close()
}

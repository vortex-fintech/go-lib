//go:build unit

package adapters

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestHTTPAdapter_ServeAndGracefulShutdown_WithListener(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) })

	srv := &http.Server{Handler: mux}
	adapter := &HTTP{Srv: srv, Lis: ln, NameStr: "http-test"}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serveErr := make(chan error, 1)
	go func() { serveErr <- adapter.Serve(ctx) }()

	time.Sleep(50 * time.Millisecond)

	client := http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get("http://" + ln.Addr().String() + "/ok")
	if err != nil {
		t.Fatalf("http get: %v", err)
	}
	_ = resp.Body.Close()

	shCtx, shCancel := context.WithTimeout(context.Background(), time.Second)
	defer shCancel()
	if err := adapter.GracefulStopWithTimeout(shCtx); err != nil {
		t.Fatalf("graceful shutdown: %v", err)
	}

	select {
	case <-serveErr:
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not exit after Shutdown")
	}
}

func TestHTTPAdapter_ForceStop(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	srv := &http.Server{Handler: http.NewServeMux()}
	adapter := &HTTP{Srv: srv, Lis: ln}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- adapter.Serve(ctx) }()

	time.Sleep(50 * time.Millisecond)
	adapter.ForceStop()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not exit after ForceStop")
	}
}

func TestHTTPAdapter_ListenAndServe_NoListener_GracefulShutdown(t *testing.T) {
	t.Parallel()

	srv := &http.Server{
		Addr:    "127.0.0.1:0",
		Handler: http.NewServeMux(),
	}
	adapter := &HTTP{Srv: srv, NameStr: "http-no-listener"}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serveErr := make(chan error, 1)
	go func() { serveErr <- adapter.Serve(ctx) }()

	time.Sleep(80 * time.Millisecond)

	shCtx, shCancel := context.WithTimeout(context.Background(), time.Second)
	defer shCancel()
	if err := adapter.GracefulStopWithTimeout(shCtx); err != nil {
		t.Fatalf("graceful shutdown (no listener): %v", err)
	}

	select {
	case <-serveErr:
	case <-time.After(2 * time.Second):
		t.Fatal("Serve (no listener) did not exit after Shutdown")
	}
}

func TestHTTPAdapter_Inflight_Request_Finishes_OnGraceful(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	startedReq := make(chan struct{})
	doneReq := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-startedReq:
		default:
			close(startedReq)
		}
		time.Sleep(150 * time.Millisecond)
		_, _ = io.WriteString(w, "ok")
		close(doneReq)
	})

	srv := &http.Server{Handler: mux}
	adapter := &HTTP{Srv: srv, Lis: ln, NameStr: "http-inflight"}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serveErr := make(chan error, 1)
	go func() { serveErr <- adapter.Serve(ctx) }()

	time.Sleep(50 * time.Millisecond)

	client := http.Client{Timeout: time.Second}
	go func() {
		_, _ = client.Get("http://" + ln.Addr().String() + "/slow")
	}()

	select {
	case <-startedReq:
	case <-time.After(time.Second):
		t.Fatal("slow handler did not start in time")
	}

	shCtx, shCancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer shCancel()
	if err := adapter.GracefulStopWithTimeout(shCtx); err != nil {
		t.Fatalf("graceful shutdown: %v", err)
	}

	select {
	case <-doneReq:
	case <-time.After(time.Second):
		t.Fatal("inflight request did not finish before shutdown")
	}

	select {
	case <-serveErr:
	case <-time.After(time.Second):
		t.Fatal("Serve did not exit after graceful stop")
	}
}

func TestHTTPAdapter_DefaultName(t *testing.T) {
	t.Parallel()
	ad := &HTTP{Srv: &http.Server{}}
	if got := ad.Name(); got != "http" {
		t.Fatalf("expected default name 'http', got %q", got)
	}
}

func TestHTTPAdapter_CustomName(t *testing.T) {
	t.Parallel()
	ad := &HTTP{Srv: &http.Server{}, NameStr: "my-http"}
	if got := ad.Name(); got != "my-http" {
		t.Fatalf("expected custom name 'my-http', got %q", got)
	}
}

//go:build integration

package prommetrics_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/vortex-fintech/go-lib/runtime/metrics"
	"github.com/vortex-fintech/go-lib/runtime/shutdown"
	"github.com/vortex-fintech/go-lib/runtime/shutdown/adapters"
	"github.com/vortex-fintech/go-lib/runtime/shutdown/prommetrics"
	"google.golang.org/grpc"
)

func TestPromMetrics_Integration_WithMetricsHandler(t *testing.T) {
	reg := prometheus.NewRegistry()

	shutdownMetrics, err := prommetrics.New(reg, "testapp", "shutdown")
	if err != nil {
		t.Fatalf("prommetrics.New() error: %v", err)
	}

	handler, _ := metrics.New(metrics.Options{
		Registry: reg,
	})

	httpLis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("http listen: %v", err)
	}
	defer httpLis.Close()

	grpcLis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("grpc listen: %v", err)
	}
	defer grpcLis.Close()

	httpSrv := &http.Server{Handler: handler}
	grpcSrv := grpc.NewServer()

	m := shutdown.New(shutdown.Config{
		ShutdownTimeout: 500 * time.Millisecond,
		Metrics:         shutdownMetrics,
	})

	m.Add(&adapters.HTTP{Srv: httpSrv, Lis: httpLis, NameStr: "http-metrics"})
	m.Add(&adapters.GRPC{Srv: grpcSrv, Lis: grpcLis, NameStr: "grpc-api"})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- m.Run(ctx) }()

	time.Sleep(50 * time.Millisecond)
	cancel()

	if err := <-done; err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("metrics handler status: %d", rec.Code)
	}

	body := rec.Body.String()

	expectedMetrics := []string{
		"testapp_shutdown_graceful_stop_total",
		"testapp_shutdown_graceful_duration_seconds",
		"testapp_shutdown_server_stop_result_total",
	}

	for _, m := range expectedMetrics {
		if !strings.Contains(body, m) {
			t.Fatalf("expected metric %q in output", m)
		}
	}

	if !strings.Contains(body, `testapp_shutdown_graceful_stop_total{result="success"}`) {
		t.Fatalf("expected success metric in output")
	}
}

func TestPromMetrics_Integration_ForceStop(t *testing.T) {
	reg := prometheus.NewRegistry()

	shutdownMetrics, err := prommetrics.New(reg, "testapp", "shutdown")
	if err != nil {
		t.Fatalf("prommetrics.New() error: %v", err)
	}

	handler, _ := metrics.New(metrics.Options{
		Registry: reg,
	})

	httpLis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("http listen: %v", err)
	}
	defer httpLis.Close()

	blockCh := make(chan struct{})
	blockForever := func() <-chan struct{} { return blockCh }

	httpSrv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-blockForever()
			w.WriteHeader(http.StatusOK)
		}),
	}

	m := shutdown.New(shutdown.Config{
		ShutdownTimeout: 30 * time.Millisecond,
		Metrics:         shutdownMetrics,
	})

	m.Add(&adapters.HTTP{Srv: httpSrv, Lis: httpLis, NameStr: "slow-http"})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- m.Run(ctx) }()

	time.Sleep(20 * time.Millisecond)

	client := &http.Client{Timeout: 2 * time.Second}
	go func() {
		resp, err := client.Get("http://" + httpLis.Addr().String() + "/test")
		if err == nil {
			resp.Body.Close()
		}
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	if err := <-done; err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	if !strings.Contains(body, `testapp_shutdown_graceful_stop_total{result="force"}`) {
		t.Fatalf("expected force metric in output, got:\n%s", body)
	}

	close(blockCh)
}

func TestPromMetrics_Integration_SharedRegistry(t *testing.T) {
	reg := prometheus.NewRegistry()

	shutdownMetrics, err := prommetrics.New(reg, "app", "shutdown")
	if err != nil {
		t.Fatalf("prommetrics.New() error: %v", err)
	}

	customCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "custom_requests_total",
		Help: "Custom counter",
	})
	if err := reg.Register(customCounter); err != nil {
		t.Fatalf("register custom counter: %v", err)
	}

	handler, _ := metrics.New(metrics.Options{
		Registry: reg,
	})

	shutdownMetrics.IncStopTotal("success")
	shutdownMetrics.IncServeError("test-server")
	shutdownMetrics.ObserveGracefulDuration(100 * time.Millisecond)
	customCounter.Inc()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	expectedMetrics := []string{
		"app_shutdown_graceful_stop_total",
		"app_shutdown_server_serve_errors_total",
		"app_shutdown_graceful_duration_seconds",
		"custom_requests_total",
		"go_goroutines",
		"process_resident_memory_bytes",
	}

	for _, m := range expectedMetrics {
		if !strings.Contains(body, m) {
			t.Fatalf("expected metric %q in output", m)
		}
	}
}

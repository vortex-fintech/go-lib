//go:build unit

package prommetrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestPromMetrics_CountersAndHistogram(t *testing.T) {
	reg := prometheus.NewRegistry()
	pm, err := New(reg, "vortex", "shutdown")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	pm.IncStopTotal("success")
	pm.IncStopTotal("force")
	pm.IncStopTotal("force")

	pm.IncServeError("grpc-auth")
	pm.IncServeError("grpc-auth")
	pm.IncServerStopResult("http-metrics", "success")
	pm.IncServerStopResult("grpc-auth", "force")

	if got, want := testutil.ToFloat64(pm.stopTotal.WithLabelValues("success")), 1.0; got != want {
		t.Fatalf("stop_total{success}=%v want %v", got, want)
	}
	if got, want := testutil.ToFloat64(pm.stopTotal.WithLabelValues("force")), 2.0; got != want {
		t.Fatalf("stop_total{force}=%v want %v", got, want)
	}
	if got, want := testutil.ToFloat64(pm.serveErrors.WithLabelValues("grpc-auth")), 2.0; got != want {
		t.Fatalf("serve_errors{grpc-auth}=%v want %v", got, want)
	}
	if got, want := testutil.ToFloat64(pm.serverStopResult.WithLabelValues("http-metrics", "success")), 1.0; got != want {
		t.Fatalf("server_stop_result{http-metrics,success}=%v want %v", got, want)
	}
	if got, want := testutil.ToFloat64(pm.serverStopResult.WithLabelValues("grpc-auth", "force")), 1.0; got != want {
		t.Fatalf("server_stop_result{grpc-auth,force}=%v want %v", got, want)
	}

	pm.ObserveGracefulDuration(150 * time.Millisecond)

	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("reg.Gather err: %v", err)
	}

	var found bool
	for _, mf := range mfs {
		if mf.GetName() == "vortex_shutdown_graceful_duration_seconds" {
			found = true
			if len(mf.Metric) == 0 || mf.Metric[0].Histogram == nil || mf.Metric[0].Histogram.GetSampleCount() == 0 {
				t.Fatalf("histogram exists but sample count is zero")
			}
			break
		}
	}
	if !found {
		t.Fatalf("histogram vortex_shutdown_graceful_duration_seconds not found")
	}
}

func TestPromMetrics_NilRegistry(t *testing.T) {
	_, err := New(nil, "vortex", "shutdown")
	if err == nil {
		t.Fatal("expected error for nil registry")
	}
}

func TestPromMetrics_DoubleRegister(t *testing.T) {
	reg := prometheus.NewRegistry()

	pm1, err := New(reg, "vortex", "shutdown")
	if err != nil {
		t.Fatalf("first New() error: %v", err)
	}

	pm2, err := New(reg, "vortex", "shutdown")
	if err != nil {
		t.Fatalf("second New() should succeed with AlreadyRegistered, got: %v", err)
	}

	if pm1 != pm2 {
		t.Log("Note: different instances but metrics registered once")
	}
}

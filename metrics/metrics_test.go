package metrics_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/vortex-fintech/go-lib/metrics"
)

func TestMetricsHandler_Defaults(t *testing.T) {
	// Регистрируем простую метрику, чтобы /metrics не был пустым
	h, _ := metrics.New(metrics.Options{
		Register: func(r prometheus.Registerer) error {
			c := prometheus.NewCounter(prometheus.CounterOpts{
				Name: "test_metric_total",
				Help: "test metric to ensure output is not empty",
			})
			return r.Register(c)
		},
	})

	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	// Проверяем, что есть HELP/NAME нашей метрики
	require.Contains(t, string(body), "# HELP test_metric_total")
	require.Contains(t, string(body), "# TYPE test_metric_total counter")

	resp, err = http.Get(srv.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestMetricsHandler_CustomHealth(t *testing.T) {
	h, _ := metrics.New(metrics.Options{
		Health: func(_ context.Context, _ *http.Request) error {
			return errors.New("db down")
		},
		HealthTimeout: 50 * time.Millisecond,
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

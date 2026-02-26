package prommetrics

import (
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// PromMetrics implements shutdown.Metrics interface using Prometheus.
// Register it with your metrics handler to expose shutdown statistics.
type PromMetrics struct {
	stopTotal        *prometheus.CounterVec
	serveErrors      *prometheus.CounterVec
	serverStopResult *prometheus.CounterVec
	gracefulDuration prometheus.Histogram
}

func registerCollector(reg prometheus.Registerer, c prometheus.Collector) error {
	if err := reg.Register(c); err != nil {
		var are prometheus.AlreadyRegisteredError
		if errors.As(err, &are) {
			return nil
		}
		return fmt.Errorf("register collector: %w", err)
	}
	return nil
}

// New creates a PromMetrics instance and registers all metrics with the provided registry.
// Namespace and subsystem are used as prefixes for metric names.
//
// Metrics registered:
//   - {namespace}_{subsystem}_graceful_stop_total{result} - counter of shutdowns by result (success/force)
//   - {namespace}_{subsystem}_server_serve_errors_total{name} - counter of non-normal serve errors
//   - {namespace}_{subsystem}_server_stop_result_total{name, result} - per-server stop result
//   - {namespace}_{subsystem}_graceful_duration_seconds - histogram of shutdown duration
//
// Returns error if reg is nil or if registration fails (except AlreadyRegisteredError).
func New(reg prometheus.Registerer, namespace, subsystem string) (*PromMetrics, error) {
	if reg == nil {
		return nil, errors.New("prometheus registerer is nil")
	}

	hist := prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace, Subsystem: subsystem,
		Name:    "graceful_duration_seconds",
		Help:    "Duration of global graceful stop",
		Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 20, 30, 60},
	})

	pm := &PromMetrics{
		stopTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Subsystem: subsystem,
			Name: "graceful_stop_total", Help: "Total graceful stops by result",
		}, []string{"result"}),

		serveErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Subsystem: subsystem,
			Name: "server_serve_errors_total", Help: "Non-normal serve() errors by server name",
		}, []string{"name"}),

		gracefulDuration: hist,

		serverStopResult: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Subsystem: subsystem,
			Name: "server_stop_result_total", Help: "Per-server graceful stop result",
		}, []string{"name", "result"}),
	}

	for _, c := range []prometheus.Collector{pm.stopTotal, pm.serveErrors, pm.serverStopResult, hist} {
		if err := registerCollector(reg, c); err != nil {
			return nil, err
		}
	}

	return pm, nil
}

func (p *PromMetrics) IncStopTotal(result string) {
	p.stopTotal.WithLabelValues(result).Inc()
}

func (p *PromMetrics) ObserveGracefulDuration(d time.Duration) {
	p.gracefulDuration.Observe(d.Seconds())
}

func (p *PromMetrics) IncServeError(name string) {
	p.serveErrors.WithLabelValues(name).Inc()
}

func (p *PromMetrics) IncServerStopResult(name, result string) {
	p.serverStopResult.WithLabelValues(name, result).Inc()
}

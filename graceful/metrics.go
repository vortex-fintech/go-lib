package graceful

import (
	"errors"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type PromMetrics struct {
	stopTotal        *prometheus.CounterVec
	serveErrors      *prometheus.CounterVec
	serverStopResult *prometheus.CounterVec
	gracefulDuration prometheus.Observer
}

func registerCollector(reg prometheus.Registerer, c prometheus.Collector) {
	if err := reg.Register(c); err != nil {
		var are prometheus.AlreadyRegisteredError
		if errors.As(err, &are) {
			return
		}
	}
}

func NewPromMetrics(reg prometheus.Registerer, namespace, subsystem string) *PromMetrics {
	pm := &PromMetrics{
		stopTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Subsystem: subsystem,
			Name: "graceful_stop_total", Help: "Total graceful stops by result",
		}, []string{"result"}),

		serveErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Subsystem: subsystem,
			Name: "server_serve_errors_total", Help: "Non-normal serve() errors by server name",
		}, []string{"name"}),

		gracefulDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace, Subsystem: subsystem,
			Name: "graceful_duration_seconds", Help: "Duration of global graceful stop",
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 20, 30, 60},
		}),

		serverStopResult: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace, Subsystem: subsystem,
			Name: "server_stop_result_total", Help: "Per-server graceful stop result",
		}, []string{"name", "result"}),
	}

	registerCollector(reg, pm.stopTotal)
	registerCollector(reg, pm.serveErrors)
	registerCollector(reg, pm.serverStopResult)
	registerCollector(reg, pm.gracefulDuration.(prometheus.Collector))

	return pm
}

func (p *PromMetrics) IncStopTotal(result string) {
	p.stopTotal.WithLabelValues(result).Inc()
}

func (p *PromMetrics) ObserveGracefulDuration(d time.Duration) {
	p.gracefulDuration.(prometheus.Histogram).Observe(d.Seconds())
}

func (p *PromMetrics) IncServeError(name string) {
	p.serveErrors.WithLabelValues(name).Inc()
}

func (p *PromMetrics) IncServerStopResult(name, result string) {
	p.serverStopResult.WithLabelValues(name, result).Inc()
}

package metrics

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Options настраивает /metrics и /health.
type Options struct {
	Registry      *prometheus.Registry
	Register      func(reg prometheus.Registerer) error
	Health        func(ctx context.Context, r *http.Request) error
	MetricsPath   string
	HealthPath    string
	HealthTimeout time.Duration
}

func registerCollector(reg prometheus.Registerer, c prometheus.Collector) {
	if err := reg.Register(c); err != nil {
		var are prometheus.AlreadyRegisteredError
		if errors.As(err, &are) {
			// Уже зарегистрирован — ок.
			return
		}
		// Иные ошибки игнорируем (или залогируйте снаружи).
	}
}

// New создаёт http.Handler для /metrics и /health и возвращает (handler, registry).
func New(opts Options) (http.Handler, *prometheus.Registry) {
	if opts.MetricsPath == "" {
		opts.MetricsPath = "/metrics"
	}
	if opts.HealthPath == "" {
		opts.HealthPath = "/health"
	}
	if opts.HealthTimeout <= 0 {
		opts.HealthTimeout = 500 * time.Millisecond
	}

	reg := opts.Registry
	if reg == nil {
		reg = prometheus.NewRegistry()
	}

	// Стандартные метрики процесса/рантайма.
	registerCollector(reg, prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	registerCollector(reg, prometheus.NewGoCollector())

	// Бизнес-метрики сервиса.
	if opts.Register != nil {
		_ = opts.Register(reg)
	}

	mux := http.NewServeMux()

	// /metrics
	mux.Handle(opts.MetricsPath, promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	// /health с форс-таймаутом.
	mux.HandleFunc(opts.HealthPath, func(w http.ResponseWriter, r *http.Request) {
		if opts.Health == nil {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), opts.HealthTimeout)
		defer cancel()

		errCh := make(chan error, 1)
		go func() { errCh <- opts.Health(ctx, r) }()

		select {
		case err := <-errCh:
			if err != nil {
				http.Error(w, "UNHEALTHY: "+err.Error(), http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		case <-ctx.Done():
			http.Error(w, "UNHEALTHY: health timeout", http.StatusServiceUnavailable)
		}
	})

	return mux, reg
}

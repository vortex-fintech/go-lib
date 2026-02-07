package metrics

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const healthCheckConcurrencyLimit = 64

// Options определяет, как сконфигурировать хендлеры /metrics и /health.
type Options struct {
	// Registry: если nil — будет создан свой prometheus.NewRegistry().
	// ВАЖНО: используем *prometheus.Registry, т.к. он реализует и Registerer, и Gatherer.
	Registry *prometheus.Registry

	// Register вызывается после регистрации стандартных метрик.
	// Здесь можно регать бизнес-метрики. Возвращаемую ошибку мы намеренно игнорируем,
	// чтобы не паниковать/не падать при AlreadyRegistered и прочем — ответственность на вызывающем.
	Register func(reg prometheus.Registerer) error

	// Health — проверка готовности. Если nil — /health всегда OK.
	// Если задано — вызывается в отдельной горутине с таймаутом HealthTimeout.
	// При ошибке или таймауте возвращаем 503.
	Health func(ctx context.Context, r *http.Request) error

	// Пути (дефолты см. ниже).
	MetricsPath string
	HealthPath  string

	// Таймаут на Health (по умолчанию 500ms).
	HealthTimeout time.Duration
}

// registerCollector регистрирует коллектор и безопасно игнорирует AlreadyRegistered.
func registerCollector(reg prometheus.Registerer, c prometheus.Collector) {
	if err := reg.Register(c); err != nil {
		var are prometheus.AlreadyRegisteredError
		if errors.As(err, &are) {
			return
		}
	}
}

// New собирает mux с /metrics и /health и возвращает его вместе с реестром метрик.
func New(opts Options) (http.Handler, *prometheus.Registry) {
	metricsPath := opts.MetricsPath
	if metricsPath == "" {
		metricsPath = "/metrics"
	}
	healthPath := opts.HealthPath
	if healthPath == "" {
		healthPath = "/health"
	}
	ht := opts.HealthTimeout
	if ht <= 0 {
		ht = 500 * time.Millisecond
	}
	reg := opts.Registry
	if reg == nil {
		reg = prometheus.NewRegistry()
	}

	// Регистрируем стандартные метрики процесса и рантайма.
	// Используем registerCollector, чтобы молча игнорировать AlreadyRegistered.
	registerCollector(reg, collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	registerCollector(reg, collectors.NewGoCollector())

	// Пользовательские метрики.
	if opts.Register != nil {
		_ = opts.Register(reg)
	}

	mux := http.NewServeMux()
	healthSem := make(chan struct{}, healthCheckConcurrencyLimit)

	// /metrics — пускаем только GET/HEAD.
	mux.Handle(metricsPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		promhttp.HandlerFor(reg, promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		}).ServeHTTP(w, r)
	}))

	// /health — если Health не задан, возвращаем OK; иначе уважаем таймаут.
	mux.Handle(healthPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if opts.Health == nil {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), ht)
		defer cancel()

		select {
		case healthSem <- struct{}{}:
		default:
			http.Error(w, "health check busy", http.StatusServiceUnavailable)
			return
		}

		done := make(chan error, 1)
		go func() {
			defer func() { <-healthSem }()
			done <- opts.Health(ctx, r)
		}()

		select {
		case err := <-done:
			if err != nil {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		case <-ctx.Done():
			http.Error(w, "health check timeout", http.StatusServiceUnavailable)
		}
	}))

	return mux, reg
}

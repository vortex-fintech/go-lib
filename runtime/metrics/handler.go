package metrics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const healthCheckConcurrencyLimit = 64

type LogLevel string

const (
	LogDebug LogLevel = "DEBUG"
	LogInfo  LogLevel = "INFO"
	LogWarn  LogLevel = "WARN"
	LogError LogLevel = "ERROR"
)

type LogFunc func(level LogLevel, path, method string, status int, duration time.Duration)

type AuthFunc func(r *http.Request) bool

type Options struct {
	Registry *prometheus.Registry
	Register func(reg prometheus.Registerer) error

	// Health and Ready must respect ctx.Done() and return promptly on cancellation,
	// otherwise healthCheckConcurrencyLimit can be exhausted by stuck checks.
	Health func(ctx context.Context, r *http.Request) error
	Ready  func(ctx context.Context, r *http.Request) error

	MetricsPath string
	HealthPath  string
	ReadyPath   string

	HealthTimeout time.Duration
	ReadyTimeout  time.Duration

	MetricsAuth AuthFunc
	Log         LogFunc

	// StrictRegister: if true, New() returns (nil, nil) when metric registration fails.
	// Always check handler != nil when using StrictRegister.
	// NOTE: if Log is nil, failure reason is not recorded.
	StrictRegister bool

	// DisableBuildInfo: if true, does not register build_info metrics.
	DisableBuildInfo bool
}

func registerCollector(reg prometheus.Registerer, c prometheus.Collector, log LogFunc, name string) error {
	if err := reg.Register(c); err != nil {
		var are prometheus.AlreadyRegisteredError
		if errors.As(err, &are) {
			return nil
		}
		if log != nil {
			log(LogError, fmt.Sprintf("metrics.register.%s: %v", name, err), "REGISTER", http.StatusInternalServerError, 0)
		}
		return err
	}
	return nil
}

func methodNotAllowed(w http.ResponseWriter, headOnly bool) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Allow", "GET, HEAD")
	writeError(w, "method not allowed", http.StatusMethodNotAllowed, headOnly)
}

func normalizePath(p, def string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		p = strings.TrimSpace(def)
	}
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	return p
}

func writeError(w http.ResponseWriter, msg string, status int, headOnly bool) {
	if headOnly {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(status)
		return
	}
	http.Error(w, msg, status)
}

func New(opts Options) (http.Handler, *prometheus.Registry) {
	metricsPath := normalizePath(opts.MetricsPath, "/metrics")
	healthPath := normalizePath(opts.HealthPath, "/health")
	readyPath := normalizePath(opts.ReadyPath, "/ready")

	healthTimeout := opts.HealthTimeout
	if healthTimeout <= 0 {
		healthTimeout = 500 * time.Millisecond
	}
	readyTimeout := opts.ReadyTimeout
	if readyTimeout <= 0 {
		readyTimeout = 500 * time.Millisecond
	}

	reg := opts.Registry
	if reg == nil {
		reg = prometheus.NewRegistry()
	}

	log := opts.Log
	strict := opts.StrictRegister

	if err := registerCollector(reg, collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}), log, "process"); err != nil && strict {
		return nil, nil
	}
	if err := registerCollector(reg, collectors.NewGoCollector(), log, "go"); err != nil && strict {
		return nil, nil
	}
	if !opts.DisableBuildInfo {
		if err := registerCollector(reg, collectors.NewBuildInfoCollector(), log, "build_info"); err != nil && strict {
			return nil, nil
		}
	}

	if opts.Register != nil {
		if err := opts.Register(reg); err != nil {
			if log != nil {
				log(LogError, fmt.Sprintf("metrics.register.custom: %v", err), "REGISTER", http.StatusInternalServerError, 0)
			}
			if strict {
				return nil, nil
			}
		}
	}

	mux := http.NewServeMux()
	healthSem := make(chan struct{}, healthCheckConcurrencyLimit)

	metricsHandler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})

	mux.Handle(metricsPath, withLog(
		withMetricsAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				methodNotAllowed(w, r.Method == http.MethodHead)
				return
			}
			w.Header().Set("Cache-Control", "no-store")
			metricsHandler.ServeHTTP(w, r)
		}), opts.MetricsAuth),
		metricsPath, log,
	))

	mux.Handle(healthPath, withLog(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			methodNotAllowed(w, r.Method == http.MethodHead)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		runHealthCheck(w, r, opts.Health, healthTimeout, healthSem, r.Method == http.MethodHead)
	}), healthPath, log))

	mux.Handle(readyPath, withLog(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			methodNotAllowed(w, r.Method == http.MethodHead)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		runHealthCheck(w, r, opts.Ready, readyTimeout, healthSem, r.Method == http.MethodHead)
	}), readyPath, log))

	return mux, reg
}

func runHealthCheck(w http.ResponseWriter, r *http.Request, check func(context.Context, *http.Request) error, timeout time.Duration, sem chan struct{}, headOnly bool) {
	if check == nil {
		w.WriteHeader(http.StatusOK)
		if !headOnly {
			_, _ = w.Write([]byte("OK"))
		}
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	select {
	case sem <- struct{}{}:
	default:
		w.Header().Set("Retry-After", "1")
		writeError(w, "health check busy", http.StatusServiceUnavailable, headOnly)
		return
	}

	done := make(chan error, 1)
	go func() {
		defer func() { <-sem }()
		done <- check(ctx, r)
	}()

	select {
	case err := <-done:
		if err != nil {
			writeError(w, err.Error(), http.StatusServiceUnavailable, headOnly)
			return
		}
		w.WriteHeader(http.StatusOK)
		if !headOnly {
			_, _ = w.Write([]byte("OK"))
		}
	case <-ctx.Done():
		w.Header().Set("Retry-After", "1")
		writeError(w, "health check timeout", http.StatusServiceUnavailable, headOnly)
	}
}

func withLog(h http.Handler, path string, log LogFunc) http.Handler {
	if log == nil {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w}
		h.ServeHTTP(lrw, r)
		if lrw.status == 0 {
			lrw.status = http.StatusOK
		}
		log(logLevelFromStatus(lrw.status), path, r.Method, lrw.status, time.Since(start))
	})
}

func withMetricsAuth(h http.Handler, auth AuthFunc) http.Handler {
	if auth == nil {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !auth(r) {
			writeError(w, "unauthorized", http.StatusUnauthorized, r.Method == http.MethodHead)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func logLevelFromStatus(status int) LogLevel {
	switch {
	case status < 400:
		return LogInfo
	case status < 500:
		return LogWarn
	default:
		return LogError
	}
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (l *loggingResponseWriter) WriteHeader(statusCode int) {
	l.status = statusCode
	l.ResponseWriter.WriteHeader(statusCode)
}

func (l *loggingResponseWriter) Write(p []byte) (int, error) {
	if l.status == 0 {
		l.status = http.StatusOK
	}
	return l.ResponseWriter.Write(p)
}

func (l *loggingResponseWriter) Flush() {
	if f, ok := l.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (l *loggingResponseWriter) Unwrap() http.ResponseWriter {
	return l.ResponseWriter
}

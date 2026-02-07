package metrics

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestMetricsHandler_Defaults(t *testing.T) {
	t.Parallel()

	// Добавим пользовательскую метрику, чтобы проверить экспозицию
	var ctr = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "test",
		Name:      "metric_total",
		Help:      "test counter",
	})

	h, _ := New(Options{
		Register: func(reg prometheus.Registerer) error {
			return reg.Register(ctr)
		},
	})

	srv := httptest.NewServer(h)
	defer srv.Close()

	// /metrics
	resp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status /metrics = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	content := string(body)
	if !strings.Contains(content, "# HELP test_metric_total test counter") {
		t.Fatalf("metrics output missing HELP line:\n%s", content)
	}
	if !strings.Contains(content, "# TYPE test_metric_total counter") {
		t.Fatalf("metrics output missing TYPE line:\n%s", content)
	}

	// /health (по умолчанию OK)
	hr, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer hr.Body.Close()
	if hr.StatusCode != http.StatusOK {
		t.Fatalf("status /health = %d, want 200", hr.StatusCode)
	}
}

func TestMetricsHandler_CustomHealth(t *testing.T) {
	t.Parallel()

	h, _ := New(Options{
		HealthTimeout: 50 * time.Millisecond,
		Health: func(ctx context.Context, r *http.Request) error {
			return errors.New("db down")
		},
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status /health = %d, want 503", resp.StatusCode)
	}
}

func TestMetricsHandler_HealthTimeout_Returns503(t *testing.T) {
	t.Parallel()

	h, _ := New(Options{
		HealthTimeout: 50 * time.Millisecond,
		// Эмулируем зависание (не уважаем контекст), чтобы сработала ветка таймаута.
		Health: func(ctx context.Context, r *http.Request) error {
			time.Sleep(200 * time.Millisecond)
			return nil
		},
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status /health = %d, want 503 (timeout)", resp.StatusCode)
	}
}

func TestMetricsHandler_CustomPaths(t *testing.T) {
	t.Parallel()

	var ctr = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "x_metric_total",
		Help: "x",
	})

	h, _ := New(Options{
		MetricsPath: "/m",
		HealthPath:  "/h",
		Register: func(reg prometheus.Registerer) error {
			return reg.Register(ctr)
		},
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	// /m
	resp, err := http.Get(srv.URL + "/m")
	if err != nil {
		t.Fatalf("GET /m: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status /m = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "# TYPE x_metric_total counter") {
		t.Fatalf("metrics output missing our counter:\n%s", string(body))
	}

	// /h
	hr, err := http.Get(srv.URL + "/h")
	if err != nil {
		t.Fatalf("GET /h: %v", err)
	}
	defer hr.Body.Close()
	if hr.StatusCode != http.StatusOK {
		t.Fatalf("status /h = %d, want 200", hr.StatusCode)
	}
}

func TestMetricsHandler_ReuseRegistry_AlreadyRegistered_NoPanic(t *testing.T) {
	t.Parallel()

	reg := prometheus.NewRegistry()
	ctr := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "dup",
		Name:      "demo_total",
		Help:      "dup",
	})

	// Первый хендлер: регистрируем counter
	h1, _ := New(Options{
		Registry: reg,
		Register: func(r prometheus.Registerer) error {
			return r.Register(ctr)
		},
	})
	s1 := httptest.NewServer(h1)
	defer s1.Close()

	// Второй хендлер с тем же регистром и той же метрикой — должно быть ок (AlreadyRegistered).
	h2, _ := New(Options{
		Registry: reg,
		Register: func(r prometheus.Registerer) error {
			// Попытка повторной регистрации вернёт AlreadyRegisteredError
			return r.Register(ctr)
		},
	})
	s2 := httptest.NewServer(h2)
	defer s2.Close()

	// Оба эндпоинта должны работать и содержать нашу метрику.
	check := func(url string) {
		resp, err := http.Get(url + "/metrics")
		if err != nil {
			t.Fatalf("GET /metrics: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status /metrics = %d, want 200", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		out := string(body)
		if !strings.Contains(out, "# TYPE dup_demo_total counter") {
			t.Fatalf("metrics missing dup_demo_total:\n%s", out)
		}
	}
	check(s1.URL)
	check(s2.URL)
}

func TestMetricsHandler_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	h, _ := New(Options{})
	s := httptest.NewServer(h)
	defer s.Close()

	req, _ := http.NewRequest(http.MethodPost, s.URL+"/metrics", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /metrics: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status /metrics = %d, want 405", resp.StatusCode)
	}

	req2, _ := http.NewRequest(http.MethodPut, s.URL+"/health", nil)
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("PUT /health: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status /health = %d, want 405", resp2.StatusCode)
	}
}

func TestMetricsHandler_HealthConcurrencyLimit(t *testing.T) {
	t.Parallel()

	var maxInFlight int32
	var current int32

	h, _ := New(Options{
		HealthTimeout: time.Second,
		Health: func(ctx context.Context, r *http.Request) error {
			n := atomic.AddInt32(&current, 1)
			for {
				m := atomic.LoadInt32(&maxInFlight)
				if n <= m || atomic.CompareAndSwapInt32(&maxInFlight, m, n) {
					break
				}
			}
			time.Sleep(200 * time.Millisecond)
			atomic.AddInt32(&current, -1)
			return nil
		},
	})

	srv := httptest.NewServer(h)
	defer srv.Close()

	const total = healthCheckConcurrencyLimit + 20
	var wg sync.WaitGroup
	var unavailable int32

	for i := 0; i < total; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(srv.URL + "/health")
			if err != nil {
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusServiceUnavailable {
				atomic.AddInt32(&unavailable, 1)
			}
		}()
	}

	wg.Wait()

	if atomic.LoadInt32(&maxInFlight) > healthCheckConcurrencyLimit {
		t.Fatalf("max in-flight checks = %d, limit = %d", maxInFlight, healthCheckConcurrencyLimit)
	}
	if atomic.LoadInt32(&unavailable) == 0 {
		t.Fatalf("expected some requests to be limited with 503")
	}
}

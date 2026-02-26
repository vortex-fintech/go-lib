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

func TestMetricsHandler_ReadyEndpoint(t *testing.T) {
	t.Parallel()

	h, _ := New(Options{
		Ready: func(ctx context.Context, r *http.Request) error {
			return errors.New("cache not warmed")
		},
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/ready")
	if err != nil {
		t.Fatalf("GET /ready: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status /ready = %d, want 503", resp.StatusCode)
	}
}

func TestMetricsHandler_ReadyEndpoint_OK(t *testing.T) {
	t.Parallel()

	h, _ := New(Options{
		Ready: func(ctx context.Context, r *http.Request) error {
			return nil
		},
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/ready")
	if err != nil {
		t.Fatalf("GET /ready: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status /ready = %d, want 200", resp.StatusCode)
	}
}

func TestMetricsHandler_Auth(t *testing.T) {
	t.Parallel()

	h, _ := New(Options{
		MetricsAuth: func(r *http.Request) bool {
			return r.Header.Get("Authorization") == "Bearer secret"
		},
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status /metrics without auth = %d, want 401", resp.StatusCode)
	}

	req, _ := http.NewRequest("GET", srv.URL+"/metrics", nil)
	req.Header.Set("Authorization", "Bearer secret")
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /metrics with auth: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("status /metrics with auth = %d, want 200", resp2.StatusCode)
	}
}

func TestMetricsHandler_Logging(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var logs []struct {
		level    LogLevel
		path     string
		method   string
		status   int
		duration time.Duration
	}

	h, _ := New(Options{
		Log: func(level LogLevel, path, method string, status int, duration time.Duration) {
			mu.Lock()
			logs = append(logs, struct {
				level    LogLevel
				path     string
				method   string
				status   int
				duration time.Duration
			}{level, path, method, status, duration})
			mu.Unlock()
		},
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	resp.Body.Close()

	resp2, err := http.Post(srv.URL+"/health", "", nil)
	if err != nil {
		t.Fatalf("POST /health: %v", err)
	}
	resp2.Body.Close()

	mu.Lock()
	defer mu.Unlock()

	if len(logs) != 2 {
		t.Fatalf("expected 2 log entries, got %d", len(logs))
	}

	if logs[0].path != "/health" || logs[0].status != http.StatusOK {
		t.Fatalf("first log: path=%s status=%d, want /health 200", logs[0].path, logs[0].status)
	}
	if logs[1].status != http.StatusMethodNotAllowed {
		t.Fatalf("second log: status=%d, want 405", logs[1].status)
	}
}

func TestMetricsHandler_HealthAndReadySeparate(t *testing.T) {
	t.Parallel()

	var healthCalled, readyCalled bool

	h, _ := New(Options{
		Health: func(ctx context.Context, r *http.Request) error {
			healthCalled = true
			return nil
		},
		Ready: func(ctx context.Context, r *http.Request) error {
			readyCalled = true
			return nil
		},
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	http.Get(srv.URL + "/health")
	http.Get(srv.URL + "/ready")

	if !healthCalled {
		t.Fatal("health check was not called")
	}
	if !readyCalled {
		t.Fatal("ready check was not called")
	}
}

func TestMetricsHandler_HeadNoBody(t *testing.T) {
	t.Parallel()

	h, _ := New(Options{
		Health: func(ctx context.Context, r *http.Request) error { return nil },
		Ready:  func(ctx context.Context, r *http.Request) error { return nil },
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodHead, srv.URL+"/health", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HEAD /health: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if len(body) > 0 {
		t.Fatalf("HEAD /health returned body, want empty")
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	req2, _ := http.NewRequest(http.MethodHead, srv.URL+"/ready", nil)
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("HEAD /ready: %v", err)
	}
	defer resp2.Body.Close()
	body2, _ := io.ReadAll(resp2.Body)
	if len(body2) > 0 {
		t.Fatalf("HEAD /ready returned body, want empty")
	}
}

func TestMetricsHandler_Auth401Logged(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var loggedStatus int

	h, _ := New(Options{
		MetricsAuth: func(r *http.Request) bool { return false },
		Log: func(level LogLevel, path, method string, status int, duration time.Duration) {
			mu.Lock()
			loggedStatus = status
			mu.Unlock()
		},
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	resp.Body.Close()

	mu.Lock()
	status := loggedStatus
	mu.Unlock()

	if status != http.StatusUnauthorized {
		t.Fatalf("logged status = %d, want 401", status)
	}
}

func TestMetricsHandler_CacheControlNoStore(t *testing.T) {
	t.Parallel()

	h, _ := New(Options{})
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close()

	cc := resp.Header.Get("Cache-Control")
	if cc != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", cc)
	}
}

func TestMetricsHandler_RetryAfterOnBusy(t *testing.T) {
	t.Parallel()

	blockCh := make(chan struct{})
	h, _ := New(Options{
		HealthTimeout: time.Second,
		Health: func(ctx context.Context, r *http.Request) error {
			<-blockCh
			return nil
		},
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	const numRequests = healthCheckConcurrencyLimit + 5
	var wg sync.WaitGroup
	var retryAfterFound atomic.Bool

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(srv.URL + "/health")
			if err != nil {
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusServiceUnavailable {
				if resp.Header.Get("Retry-After") == "1" {
					retryAfterFound.Store(true)
				}
			}
		}()
		time.Sleep(2 * time.Millisecond)
	}

	time.Sleep(100 * time.Millisecond)
	close(blockCh)
	wg.Wait()

	if !retryAfterFound.Load() {
		t.Fatal("expected Retry-After header on 503 response")
	}
}

func TestMetricsHandler_RegisterErrorLogged(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var loggedPath string

	_, _ = New(Options{
		Register: func(reg prometheus.Registerer) error {
			return errors.New("registration failed")
		},
		Log: func(level LogLevel, path, method string, status int, duration time.Duration) {
			mu.Lock()
			loggedPath = path
			mu.Unlock()
		},
	})

	mu.Lock()
	path := loggedPath
	mu.Unlock()

	if !strings.Contains(path, "metrics.register.custom") {
		t.Fatalf("logged path = %q, want to contain 'metrics.register.custom'", path)
	}
	if !strings.Contains(path, "registration failed") {
		t.Fatalf("logged path = %q, want to contain error message", path)
	}
}

func TestMetricsHandler_StrictRegister_ReturnsNil(t *testing.T) {
	t.Parallel()

	h, reg := New(Options{
		StrictRegister: true,
		Register: func(reg prometheus.Registerer) error {
			return errors.New("registration failed")
		},
	})

	if h != nil || reg != nil {
		t.Fatal("expected nil handler and registry on StrictRegister failure")
	}
}

func TestMetricsHandler_StrictRegister_OK(t *testing.T) {
	t.Parallel()

	h, reg := New(Options{
		StrictRegister: true,
		Register: func(reg prometheus.Registerer) error {
			return nil
		},
	})

	if h == nil || reg == nil {
		t.Fatal("expected non-nil handler and registry on success")
	}
}

func TestMetricsHandler_CacheControlOnHealthAndReady(t *testing.T) {
	t.Parallel()

	h, _ := New(Options{
		Health: func(ctx context.Context, r *http.Request) error { return nil },
		Ready:  func(ctx context.Context, r *http.Request) error { return nil },
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp1, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	resp1.Body.Close()
	if cc := resp1.Header.Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control /health = %q, want no-store", cc)
	}

	resp2, err := http.Get(srv.URL + "/ready")
	if err != nil {
		t.Fatalf("GET /ready: %v", err)
	}
	resp2.Body.Close()
	if cc := resp2.Header.Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control /ready = %q, want no-store", cc)
	}
}

func TestMetricsHandler_AllowHeaderOn405(t *testing.T) {
	t.Parallel()

	h, _ := New(Options{})
	srv := httptest.NewServer(h)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/metrics", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
	if allow := resp.Header.Get("Allow"); allow != "GET, HEAD" {
		t.Fatalf("Allow = %q, want 'GET, HEAD'", allow)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", cc)
	}
}

func TestMetricsHandler_PathNormalization(t *testing.T) {
	t.Parallel()

	h, _ := New(Options{
		MetricsPath: "m",
		HealthPath:  "h",
		ReadyPath:   "r",
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp1, err := http.Get(srv.URL + "/m")
	if err != nil {
		t.Fatalf("GET /m: %v", err)
	}
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("status /m = %d, want 200", resp1.StatusCode)
	}

	resp2, err := http.Get(srv.URL + "/h")
	if err != nil {
		t.Fatalf("GET /h: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("status /h = %d, want 200", resp2.StatusCode)
	}

	resp3, err := http.Get(srv.URL + "/r")
	if err != nil {
		t.Fatalf("GET /r: %v", err)
	}
	resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("status /r = %d, want 200", resp3.StatusCode)
	}
}

func TestMetricsHandler_MethodNotAllowedBody(t *testing.T) {
	t.Parallel()

	h, _ := New(Options{})
	srv := httptest.NewServer(h)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/health", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /health: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "method not allowed") {
		t.Fatalf("body = %q, want to contain 'method not allowed'", string(body))
	}
}

func TestMetricsHandler_PathTrimSpace(t *testing.T) {
	t.Parallel()

	h, _ := New(Options{
		MetricsPath: "  metrics  ",
		HealthPath:  "  health  ",
		ReadyPath:   "  ready  ",
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp1, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("status /metrics = %d, want 200", resp1.StatusCode)
	}

	resp2, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("status /health = %d, want 200", resp2.StatusCode)
	}
}

func TestMetricsHandler_HeadErrorNoBody(t *testing.T) {
	t.Parallel()

	h, _ := New(Options{
		Health: func(ctx context.Context, r *http.Request) error {
			return errors.New("db down")
		},
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodHead, srv.URL+"/health", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HEAD /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) > 0 {
		t.Fatalf("HEAD error returned body = %q, want empty", string(body))
	}
}

func TestMetricsHandler_BuildInfoCollector(t *testing.T) {
	t.Parallel()

	h, _ := New(Options{})
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	content := string(body)

	if !strings.Contains(content, "go_build_info") {
		t.Fatalf("expected go_build_info in metrics output, got:\n%s", content[:min(500, len(content))])
	}
}

func TestMetricsHandler_DisableBuildInfo(t *testing.T) {
	t.Parallel()

	h, _ := New(Options{DisableBuildInfo: true})
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	content := string(body)

	if strings.Contains(content, "go_build_info") {
		t.Fatal("expected no go_build_info when DisableBuildInfo=true")
	}
}

func TestMetricsHandler_AuthHeadNoBody(t *testing.T) {
	t.Parallel()

	h, _ := New(Options{
		MetricsAuth: func(r *http.Request) bool { return false },
	})
	srv := httptest.NewServer(h)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodHead, srv.URL+"/metrics", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HEAD /metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/plain; charset=utf-8", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) > 0 {
		t.Fatalf("HEAD unauthorized returned body = %q, want empty", string(body))
	}
}

func TestMetricsHandler_TimeoutRetryAfter(t *testing.T) {
	t.Parallel()

	h, _ := New(Options{
		HealthTimeout: 20 * time.Millisecond,
		Health: func(ctx context.Context, r *http.Request) error {
			time.Sleep(100 * time.Millisecond)
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
		t.Fatalf("status = %d, want 503", resp.StatusCode)
	}
	if ra := resp.Header.Get("Retry-After"); ra != "1" {
		t.Fatalf("Retry-After = %q, want 1", ra)
	}
}

func TestNormalizePath_EmptyDefault(t *testing.T) {
	t.Parallel()

	if got := normalizePath("  ", " "); got != "/" {
		t.Fatalf("normalizePath empty = %q, want /", got)
	}
}

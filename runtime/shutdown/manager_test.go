package shutdown

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

/* ===================== Fakes & helpers ===================== */

type fakeServer struct {
	name       string
	waitForCtx bool
	serveErr   error

	graceDelay time.Duration
	graceErr   error

	stopOnce  sync.Once
	stoppedCh chan struct{}
	forced    atomic.Bool
}

func newFakeServer(name string) *fakeServer {
	return &fakeServer{name: name, stoppedCh: make(chan struct{})}
}

func (f *fakeServer) Name() string { return f.name }

func (f *fakeServer) Serve(ctx context.Context) error {
	if !f.waitForCtx {
		return f.serveErr
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-f.stoppedCh:
		return nil
	}
}

func (f *fakeServer) GracefulStopWithTimeout(ctx context.Context) error {
	if f.graceDelay > 0 {
		t := time.NewTimer(f.graceDelay)
		defer t.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
		}
	}
	f.stopOnce.Do(func() { close(f.stoppedCh) })
	return f.graceErr
}

func (f *fakeServer) ForceStop() {
	f.forced.Store(true)
	f.stopOnce.Do(func() { close(f.stoppedCh) })
}

type logEvent struct {
	level string
	msg   string
	kv    map[string]any
}
type fakeLogger struct {
	mu   sync.Mutex
	evts []logEvent
	hits int
}

func (l *fakeLogger) log(level, msg string, kv ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	m := map[string]any{}
	for i := 0; i+1 < len(kv); i += 2 {
		k, _ := kv[i].(string)
		m[k] = kv[i+1]
	}
	l.evts = append(l.evts, logEvent{level: level, msg: msg, kv: m})
	l.hits++
}
func (l *fakeLogger) lastName() (string, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i := len(l.evts) - 1; i >= 0; i-- {
		if v, ok := l.evts[i].kv["name"].(string); ok {
			return v, true
		}
	}
	return "", false
}
func (l *fakeLogger) hasEvents() bool { l.mu.Lock(); defer l.mu.Unlock(); return l.hits > 0 }

type fakeMetrics struct {
	mu sync.Mutex

	stopTotal        map[string]int
	serveErrors      map[string]int
	serverStopResult map[string]map[string]int
	durations        []time.Duration
}

func newFakeMetrics() *fakeMetrics {
	return &fakeMetrics{
		stopTotal:        map[string]int{},
		serveErrors:      map[string]int{},
		serverStopResult: map[string]map[string]int{},
	}
}
func (m *fakeMetrics) IncStopTotal(result string) {
	m.mu.Lock()
	m.stopTotal[result]++
	m.mu.Unlock()
}
func (m *fakeMetrics) ObserveGracefulDuration(d time.Duration) {
	m.mu.Lock()
	m.durations = append(m.durations, d)
	m.mu.Unlock()
}
func (m *fakeMetrics) IncServeError(name string) {
	m.mu.Lock()
	m.serveErrors[name]++
	m.mu.Unlock()
}
func (m *fakeMetrics) IncServerStopResult(name, result string) {
	m.mu.Lock()
	if _, ok := m.serverStopResult[name]; !ok {
		m.serverStopResult[name] = map[string]int{}
	}
	m.serverStopResult[name][result]++
	m.mu.Unlock()
}

/* ===================== Tests ===================== */

func Test_Run_NoServers_OK(t *testing.T) {
	t.Parallel()
	m := New(Config{ShutdownTimeout: 100 * time.Millisecond})
	if err := m.Run(context.Background()); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func Test_Run_NormalCloseIgnored_Single(t *testing.T) {
	t.Parallel()
	m := New(Config{ShutdownTimeout: 100 * time.Millisecond})
	s := newFakeServer("http-normal")
	s.waitForCtx = false
	s.serveErr = http.ErrServerClosed
	m.Add(s)
	if err := m.Run(context.Background()); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func Test_Run_NormalCloseIgnored_Multi_OthersGraceful(t *testing.T) {
	t.Parallel()
	m := New(Config{ShutdownTimeout: 300 * time.Millisecond})
	norm := newFakeServer("normal")
	norm.waitForCtx = false
	norm.serveErr = http.ErrServerClosed

	other := newFakeServer("other")
	other.waitForCtx = true
	other.graceDelay = 20 * time.Millisecond

	m.Add(norm)
	m.Add(other)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- m.Run(ctx) }()
	time.Sleep(30 * time.Millisecond)
	cancel()

	if err := <-done; err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if other.forced.Load() {
		t.Fatalf("other should stop gracefully, not force")
	}
}

func Test_Run_ReturnsNonNormalError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	m := New(Config{ShutdownTimeout: 100 * time.Millisecond})
	s := newFakeServer("bad")
	s.waitForCtx = false
	s.serveErr = want
	m.Add(s)
	if err := m.Run(context.Background()); !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

func Test_Run_GracefulOnContextCancel(t *testing.T) {
	t.Parallel()
	m := New(Config{ShutdownTimeout: 200 * time.Millisecond})
	s := newFakeServer("srv")
	s.waitForCtx = true
	m.Add(s)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- m.Run(ctx) }()
	time.Sleep(20 * time.Millisecond)
	cancel()

	if err := <-done; err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if s.forced.Load() {
		t.Fatal("unexpected force on graceful cancel")
	}
}

func Test_Stop_ForceOnTimeout(t *testing.T) {
	t.Parallel()
	m := New(Config{ShutdownTimeout: 120 * time.Millisecond})
	s := newFakeServer("slow")
	s.waitForCtx = true
	s.graceDelay = 300 * time.Millisecond
	m.Add(s)

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan error, 1)
	go func() { ch <- m.Run(ctx) }()
	time.Sleep(30 * time.Millisecond)
	cancel()

	<-ch
	if !s.forced.Load() {
		t.Fatal("expected ForceStop on graceful timeout")
	}
}

func Test_Stop_ForceWhenGracefulReturnsError(t *testing.T) {
	t.Parallel()
	met := newFakeMetrics()
	m := New(Config{ShutdownTimeout: 300 * time.Millisecond, Metrics: met})
	s := newFakeServer("err-on-stop")
	s.waitForCtx = true
	s.graceDelay = 10 * time.Millisecond
	s.graceErr = errors.New("stop failed")
	m.Add(s)

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan error, 1)
	go func() { ch <- m.Run(ctx) }()
	time.Sleep(20 * time.Millisecond)
	cancel()

	<-ch
	if !s.forced.Load() {
		t.Fatal("expected ForceStop when GracefulStopWithTimeout returns error")
	}
	if got := met.serverStopResult["err-on-stop"]["force"]; got < 1 {
		t.Fatalf("expected per-server force metric, got %d", got)
	}
	if got := met.stopTotal["force"]; got < 1 {
		t.Fatalf("expected global force metric, got %d", got)
	}
}

func Test_Metrics_AllSuccess_StopTotalSuccess(t *testing.T) {
	t.Parallel()
	met := newFakeMetrics()
	m := New(Config{ShutdownTimeout: 300 * time.Millisecond, Metrics: met})

	a := newFakeServer("a")
	a.waitForCtx = true
	a.graceDelay = 10 * time.Millisecond
	b := newFakeServer("b")
	b.waitForCtx = true
	b.graceDelay = 15 * time.Millisecond
	m.Add(a)
	m.Add(b)

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan error, 1)
	go func() { ch <- m.Run(ctx) }()
	time.Sleep(20 * time.Millisecond)
	cancel()

	if err := <-ch; err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if met.stopTotal["success"] < 1 {
		t.Fatalf("expected stopTotal{success} to be incremented")
	}
	if met.serverStopResult["a"]["success"] < 1 || met.serverStopResult["b"]["success"] < 1 {
		t.Fatalf("expected per-server success metrics")
	}
}

func Test_ShutdownTimeoutZero_ImmediateForce(t *testing.T) {
	t.Parallel()
	m := New(Config{ShutdownTimeout: 0})
	s := newFakeServer("z")
	s.waitForCtx = true
	s.graceDelay = 50 * time.Millisecond
	m.Add(s)

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan error, 1)
	go func() { ch <- m.Run(ctx) }()
	time.Sleep(10 * time.Millisecond)
	cancel()

	<-ch
	if !s.forced.Load() {
		t.Fatal("expected immediate force with zero shutdown timeout")
	}
}

func Test_SafeName_FallbackOnEmptyName(t *testing.T) {
	t.Parallel()
	fl := &fakeLogger{}
	m := New(Config{ShutdownTimeout: 100 * time.Millisecond, Logger: fl.log})

	s := newFakeServer("")
	s.waitForCtx = true
	m.Add(s)

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan error, 1)
	go func() { ch <- m.Run(ctx) }()
	time.Sleep(20 * time.Millisecond)
	cancel()
	<-ch

	if !fl.hasEvents() {
		t.Fatal("logger should be called")
	}
	if name, ok := fl.lastName(); !ok || name != "server" {
		t.Fatalf("expected fallback name 'server', got %q", name)
	}
}

func Test_ServeEndsOnCtxErr_TreatedAsNormal(t *testing.T) {
	t.Parallel()
	m := New(Config{ShutdownTimeout: 150 * time.Millisecond})
	s := newFakeServer("srv")
	s.waitForCtx = true
	m.Add(s)

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan error, 1)
	go func() { ch <- m.Run(ctx) }()
	time.Sleep(20 * time.Millisecond)
	cancel()

	if err := <-ch; err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func Test_NilHooks_NoPanic(t *testing.T) {
	t.Parallel()
	m := New(Config{ShutdownTimeout: 100 * time.Millisecond})
	s := newFakeServer("srv")
	s.waitForCtx = true
	m.Add(s)

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan error, 1)
	go func() { ch <- m.Run(ctx) }()
	time.Sleep(20 * time.Millisecond)
	cancel()
	<-ch
}

func Test_StopBeforeRun_NoPanic(t *testing.T) {
	t.Parallel()
	m := New(Config{ShutdownTimeout: 100 * time.Millisecond})
	s := newFakeServer("srv")
	s.waitForCtx = true
	m.Add(s)

	m.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan error, 1)
	go func() { ch <- m.Run(ctx) }()
	time.Sleep(20 * time.Millisecond)
	cancel()
	if err := <-ch; err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func Test_Stop_PerServerTimeouts_DoNotMislabelSuccess(t *testing.T) {
	t.Parallel()

	met := newFakeMetrics()

	// Общий таймаут 120ms
	m := New(Config{ShutdownTimeout: 120 * time.Millisecond, Metrics: met})

	// s1 успевает (graceDelay 40ms)
	s1 := newFakeServer("fast")
	s1.waitForCtx = true
	s1.graceDelay = 40 * time.Millisecond

	// s2 не успевает (graceDelay 300ms -> DEADLINE)
	s2 := newFakeServer("slow")
	s2.waitForCtx = true
	s2.graceDelay = 300 * time.Millisecond
	// имитируем, что его Graceful вернёт context deadline exceeded
	// (наш fakeServer как раз вернёт ctx.Err(), когда дедлайн истечёт)

	m.Add(s1)
	m.Add(s2)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- m.Run(ctx) }()

	// Дадим стартануть и сразу отменим, чтобы запустился Stop()
	time.Sleep(20 * time.Millisecond)
	cancel()
	<-done

	// Проверяем классификацию:
	if got := met.serverStopResult["fast"]["success"]; got < 1 {
		t.Fatalf("expected fast=success, got %d", got)
	}
	if got := met.serverStopResult["slow"]["force"]; got < 1 {
		t.Fatalf("expected slow=force, got %d", got)
	}
	// Глобально хотя бы один force → stopTotal{force} инкрементится
	if got := met.stopTotal["force"]; got < 1 {
		t.Fatalf("expected global force, got %d", got)
	}
}

// На всякий случай убеждаемся, что ошибка дедлайна действительно идёт как ошибка graceful
func Test_fakeServer_GracefulDeadlineProducesError(t *testing.T) {
	t.Parallel()
	s := newFakeServer("x")
	s.waitForCtx = true
	s.graceDelay = 200 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := s.GracefulStopWithTimeout(ctx)
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected deadline/canceled error, got %v", err)
	}
}

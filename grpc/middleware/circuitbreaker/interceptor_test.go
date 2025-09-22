package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

/* ---------- helpers ---------- */

type testLogger struct {
	infos  int32
	warns  int32
	errors int32
}

func (l *testLogger) Info(string)  { atomic.AddInt32(&l.infos, 1) }
func (l *testLogger) Warn(string)  { atomic.AddInt32(&l.warns, 1) }
func (l *testLogger) Error(string) { atomic.AddInt32(&l.errors, 1) }

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func makeCB(t *testing.T, clk *fakeClock, opts ...Option) *Interceptor {
	t.Helper()
	all := append([]Option{
		WithFailureThreshold(3),
		WithHalfOpenSuccess(2),
		WithRecoveryTimeout(5 * time.Second),
		withNow(clk.now),
		WithLogger(&testLogger{}),
	}, opts...)
	return New(all...)
}

func callUnary(t *testing.T, itc grpc.UnaryServerInterceptor, h grpc.UnaryHandler) error {
	t.Helper()
	_, err := itc(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, h)
	return err
}

func okHandler(ctx context.Context, req any) (any, error) { return nil, nil }
func errHandler(code codes.Code) grpc.UnaryHandler {
	return func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(code, "boom")
	}
}
func bizErrHandler() grpc.UnaryHandler {
	return func(ctx context.Context, req any) (any, error) {
		return nil, errors.New("business validation failed") // не gRPC status => не трипает
	}
}

/* ---------- tests ---------- */

func Test_CLOSED_to_OPEN_after_threshold(t *testing.T) {
	clk := &fakeClock{t: time.Unix(1, 0)}
	cb := makeCB(t, clk)

	// три критичных ошибки подряд
	itc := cb.Unary()
	for i := 0; i < 3; i++ {
		if err := callUnary(t, itc, errHandler(codes.Internal)); err == nil {
			t.Fatalf("expected error on call %d", i+1)
		}
	}
	if s := cb.State(); s != "open" {
		t.Fatalf("expected state=open, got %s", s)
	}
}

func Test_OPEN_blocks_until_timeout_then_HALF_OPEN_one_probe(t *testing.T) {
	clk := &fakeClock{t: time.Unix(1, 0)}
	cb := makeCB(t, clk, WithHalfOpenSuccess(2), WithRecoveryTimeout(5*time.Second))
	itc := cb.Unary()

	// Откроем брейкер
	for i := 0; i < 3; i++ {
		_ = callUnary(t, itc, errHandler(codes.Unavailable))
	}
	if cb.State() != "open" {
		t.Fatalf("expected open, got %s", cb.State())
	}

	// До таймаута — блок
	clk.advance(5*time.Second - time.Nanosecond)
	if err := callUnary(t, itc, okHandler); status.Code(err) != codes.Unavailable {
		t.Fatalf("expected Unavailable before timeout, got %v", err)
	}

	// === A) Параллельный вызов блокируется, пока первая проба не завершилась
	clk.advance(2 * time.Nanosecond) // перешли порог → HALF-OPEN, будет одна «проба»

	started := make(chan struct{})
	release := make(chan struct{})
	blockingProbe := func(ctx context.Context, req any) (any, error) {
		close(started)  // сигнализируем, что проба началась и держит inflight
		<-release       // ждём, чтобы смоделировать долгую обработку
		return nil, nil // успех
	}

	// запускаем первую пробу (держит inflight=true)
	go func() { _ = callUnary(t, itc, blockingProbe) }()
	<-started

	// параллельный вызов во время пробы — ДОЛЖЕН блокироваться
	if err := callUnary(t, itc, okHandler); status.Code(err) != codes.Unavailable {
		t.Fatalf("expected Unavailable for concurrent call during HALF-OPEN, got %v", err)
	}

	// завершаем первую пробу
	close(release)
	time.Sleep(10 * time.Millisecond) // даём интерсептору снять inflight

	if cb.State() != "half-open" {
		t.Fatalf("expected half-open after first successful probe, got %s", cb.State())
	}

	// === B) Последовательный вызов после завершения пробы должен пройти
	if err := callUnary(t, itc, okHandler); err != nil {
		t.Fatalf("expected sequential second probe to pass, got %v", err)
	}
	// Т.к. HalfOpenSuccess=2, теперь закрываемся
	if cb.State() != "closed" {
		t.Fatalf("expected closed after two successful probes, got %s", cb.State())
	}
}

func Test_HALF_OPEN_success_to_CLOSED(t *testing.T) {
	clk := &fakeClock{t: time.Unix(1, 0)}
	cb := makeCB(t, clk, WithHalfOpenSuccess(2))
	itc := cb.Unary()

	// Откроем
	for i := 0; i < 3; i++ {
		_ = callUnary(t, itc, errHandler(codes.DeadlineExceeded))
	}
	clk.advance(5 * time.Second)

	// Проба 1 — ок
	if err := callUnary(t, itc, okHandler); err != nil {
		t.Fatalf("probe 1 expected ok, got %v", err)
	}
	// Проба 2 — ок → CLOSED
	if err := callUnary(t, itc, okHandler); err != nil {
		t.Fatalf("probe 2 expected ok, got %v", err)
	}
	if cb.State() != "closed" {
		t.Fatalf("expected closed after enough successes, got %s", cb.State())
	}
}

func Test_HALF_OPEN_failure_reopens(t *testing.T) {
	clk := &fakeClock{t: time.Unix(1, 0)}
	cb := makeCB(t, clk)
	itc := cb.Unary()

	// Откроем
	for i := 0; i < 3; i++ {
		_ = callUnary(t, itc, errHandler(codes.Internal))
	}
	clk.advance(5 * time.Second)

	// Проба — ошибка → снова OPEN
	if err := callUnary(t, itc, errHandler(codes.Internal)); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal from probe, got %v", err)
	}
	if cb.State() != "open" {
		t.Fatalf("expected open after failed probe, got %s", cb.State())
	}
}

func Test_CustomTripFunc_and_BusinessErrorsIgnored(t *testing.T) {
	clk := &fakeClock{t: time.Unix(1, 0)}
	cb := makeCB(t, clk, WithTripFunc(func(c codes.Code) bool {
		// трипаем только на Unavailable
		return c == codes.Unavailable
	}))
	itc := cb.Unary()

	// Бизнес-ошибка (не status) — игнор
	_ = callUnary(t, itc, bizErrHandler())
	// Internal — игнор (по кастомному TripFunc)
	_ = callUnary(t, itc, errHandler(codes.Internal))

	// Только Unavailable триггерит
	for i := 0; i < 3; i++ {
		_ = callUnary(t, itc, errHandler(codes.Unavailable))
	}
	if cb.State() != "open" {
		t.Fatalf("expected open on three Unavailable, got %s", cb.State())
	}
}

func Test_Reset(t *testing.T) {
	clk := &fakeClock{t: time.Unix(1, 0)}
	cb := makeCB(t, clk)
	itc := cb.Unary()

	for i := 0; i < 3; i++ {
		_ = callUnary(t, itc, errHandler(codes.Unavailable))
	}
	if cb.State() != "open" {
		t.Fatalf("expected open, got %s", cb.State())
	}

	cb.Reset()
	if cb.State() != "closed" {
		t.Fatalf("expected closed after reset, got %s", cb.State())
	}
}

func Test_HALF_OPEN_allows_only_one_inflight_concurrently(t *testing.T) {
	clk := &fakeClock{t: time.Unix(1, 0)}
	cb := makeCB(t, clk)
	itc := cb.Unary()

	// Открываем
	for i := 0; i < 3; i++ {
		_ = callUnary(t, itc, errHandler(codes.Unavailable))
	}
	clk.advance(5 * time.Second)

	// Первый пробный хэндлер «зависает», чтобы смоделировать конкуренцию
	started := make(chan struct{})
	release := make(chan struct{})
	blockingHandler := func(ctx context.Context, req any) (any, error) {
		close(started)
		<-release
		return nil, nil
	}

	var wg sync.WaitGroup
	wg.Add(2)

	var err1, err2 error
	go func() {
		defer wg.Done()
		err1 = callUnary(t, itc, blockingHandler)
	}()
	<-started // первый захватил пробу

	go func() {
		defer wg.Done()
		err2 = callUnary(t, itc, okHandler) // должен получить Unavailable
	}()

	// дайте времени второму попасть в интерсептор
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	if err1 != nil {
		t.Fatalf("probe should pass, got %v", err1)
	}
	if status.Code(err2) != codes.Unavailable {
		t.Fatalf("second concurrent call should be Unavailable, got %v", err2)
	}
}

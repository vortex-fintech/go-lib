package circuitbreaker

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

/* ---------- logger (минимальный интерфейс) ---------- */

type Logger interface {
	Info(msg string)
	Warn(msg string)
	Error(msg string)
}

type nopLogger struct{}

func (nopLogger) Info(string)  {}
func (nopLogger) Warn(string)  {}
func (nopLogger) Error(string) {}

/* ---------- публичные опции ---------- */

type CBOptions struct {
	FailureThreshold int                     // N подряд критичных ошибок ⇒ OPEN
	RecoveryTimeout  time.Duration           // пауза OPEN → HALF-OPEN
	HalfOpenSuccess  int                     // M успешных тест-RPC ⇒ CLOSED
	TripFunc         func(c codes.Code) bool // какие коды считаем «сбоем»
	Logger           Logger                  // опционально
	Now              func() time.Time        // инъекция времени (для тестов)
}

/* functional options */

type Option func(*CBOptions)

func WithFailureThreshold(n int) Option {
	return func(o *CBOptions) { o.FailureThreshold = n }
}
func WithRecoveryTimeout(d time.Duration) Option {
	return func(o *CBOptions) { o.RecoveryTimeout = d }
}
func WithHalfOpenSuccess(n int) Option {
	return func(o *CBOptions) { o.HalfOpenSuccess = n }
}
func WithTripCodes(codesToTrip ...codes.Code) Option {
	set := make(map[codes.Code]struct{}, len(codesToTrip))
	for _, c := range codesToTrip {
		set[c] = struct{}{}
	}
	return func(o *CBOptions) {
		o.TripFunc = func(c codes.Code) bool {
			_, ok := set[c]
			return ok
		}
	}
}
func WithTripFunc(f func(codes.Code) bool) Option {
	return func(o *CBOptions) { o.TripFunc = f }
}
func WithLogger(l Logger) Option {
	return func(o *CBOptions) { o.Logger = l }
}
func withNow(fn func() time.Time) Option { // для тестов
	return func(o *CBOptions) { o.Now = fn }
}

/* ---------- конструктор ---------- */

func New(opts ...Option) *Interceptor {
	o := CBOptions{}
	for _, f := range opts {
		f(&o)
	}

	// значения по умолчанию
	if o.TripFunc == nil {
		o.TripFunc = func(c codes.Code) bool {
			return c == codes.Internal ||
				c == codes.Unavailable ||
				c == codes.DeadlineExceeded
		}
	}
	if o.FailureThreshold < 1 {
		o.FailureThreshold = 5
	}
	if o.HalfOpenSuccess < 1 {
		o.HalfOpenSuccess = 1
	}
	if o.RecoveryTimeout == 0 {
		o.RecoveryTimeout = 10 * time.Second
	}
	if o.Logger == nil {
		o.Logger = nopLogger{}
	}
	if o.Now == nil {
		o.Now = time.Now
	}

	return &Interceptor{
		log:   o.Logger,
		opt:   o,
		state: stateClosed,
		now:   o.Now,
	}
}

/* ---------- внутренняя реализация ---------- */

type cbState uint8

const (
	stateClosed cbState = iota
	stateOpen
	stateHalfOpen
)

type Interceptor struct {
	log Logger
	opt CBOptions

	mu            sync.Mutex
	state         cbState
	failures      int       // подряд критичных ошибок (CLOSED)
	openSince     time.Time // тайм-штамп входа в OPEN
	inflight      bool      // true ⇒ тестовый RPC уже идёт (HALF-OPEN)
	successInHalf int       // успешных RPC в HALF-OPEN

	now func() time.Time
}

/* ---------- публичное API ---------- */

func (cb *Interceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {

		// Решаем судьбу вызова
		cb.mu.Lock()
		var wasHalfOpen bool

		switch cb.state {
		case stateOpen:
			if cb.now().Sub(cb.openSince) >= cb.opt.RecoveryTimeout {
				cb.state = stateHalfOpen
				cb.inflight = true
				cb.successInHalf = 0
				cb.openSince = cb.now() // защита от зависания тест-RPC
				wasHalfOpen = true
				cb.log.Info("circuit breaker → HALF-OPEN")
			} else {
				cb.mu.Unlock()
				return nil, status.Error(codes.Unavailable, "circuit breaker open")
			}

		case stateHalfOpen:
			if cb.inflight {
				cb.mu.Unlock()
				return nil, status.Error(codes.Unavailable, "circuit breaker half-open")
			}
			cb.inflight = true
			cb.openSince = cb.now()
			wasHalfOpen = true

		case stateClosed:
			// обычная работа
		}
		cb.mu.Unlock()

		// выполняем бизнес-логику
		resp, err := handler(ctx, req)

		// пост-обработка
		if wasHalfOpen {
			cb.finishHalfOpen(err)
		} else {
			cb.afterCall(err)
		}

		return resp, err
	}
}

// Текущий state (удобно для метрик/тестов)
func (cb *Interceptor) State() string {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	switch cb.state {
	case stateClosed:
		return "closed"
	case stateOpen:
		return "open"
	case stateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Сброс в CLOSED (например, из админки)
func (cb *Interceptor) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = stateClosed
	cb.failures = 0
	cb.inflight = false
	cb.successInHalf = 0
	cb.openSince = time.Time{}
}

/* ---------- вспомогательные методы ---------- */

// Обработка результата в фазе CLOSED
func (cb *Interceptor) afterCall(err error) {
	if err == nil {
		cb.mu.Lock()
		cb.failures = 0
		cb.mu.Unlock()
		return
	}
	st, ok := status.FromError(err)
	if !ok || !cb.opt.TripFunc(st.Code()) {
		return // бизнес-ошибка — игнорируем
	}
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	if cb.failures >= cb.opt.FailureThreshold && cb.state == stateClosed {
		cb.state = stateOpen
		cb.openSince = cb.now()
		cb.log.Error("circuit breaker OPENED")
	}
}

// Обработка результата тестового RPC в фазе HALF-OPEN
func (cb *Interceptor) finishHalfOpen(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.inflight = false // тестовый вызов завершён

	if err == nil {
		cb.successInHalf++
		if cb.successInHalf >= cb.opt.HalfOpenSuccess {
			cb.state = stateClosed
			cb.failures = 0
			cb.log.Info("circuit breaker CLOSED — service recovered")
		}
		return
	}

	if st, ok := status.FromError(err); ok && cb.opt.TripFunc(st.Code()) {
		cb.state = stateOpen
		cb.openSince = cb.now()
		cb.failures = 1
		cb.log.Warn("circuit breaker RE-OPENED from half-open")
	}
}

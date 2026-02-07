package shutdown

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type Server interface {
	Serve(ctx context.Context) error
	GracefulStopWithTimeout(ctx context.Context) error
	ForceStop()
	Name() string
}

type Metrics interface {
	IncStopTotal(result string)
	ObserveGracefulDuration(d time.Duration)
	IncServeError(name string)
	IncServerStopResult(name, result string)
}

type Config struct {
	ShutdownTimeout time.Duration
	HandleSignals   bool
	IsNormalError   func(error) bool
	Logger          func(level, msg string, kv ...any)
	Metrics         Metrics
}

type Manager struct {
	cfg     Config
	mu      sync.Mutex
	servers []Server
	stopped bool
}

func New(cfg Config) *Manager {
	if cfg.Logger == nil {
		cfg.Logger = func(level, msg string, kv ...any) { log.Printf("[%s] %s %v", level, msg, kv) }
	}
	if cfg.IsNormalError == nil {
		cfg.IsNormalError = DefaultIsNormalErr
	}
	return &Manager{cfg: cfg}
}

func (m *Manager) Add(s Server) { m.servers = append(m.servers, s) }

func (m *Manager) Run(ctx context.Context) error {
	if m.cfg.HandleSignals {
		var stop context.CancelFunc
		ctx, stop = signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
		defer stop()
	}

	g, gctx := errgroup.WithContext(ctx)
	for _, s := range m.servers {
		srv := s
		g.Go(func() error {
			name := safeName(srv)
			m.cfg.Logger("INFO", "serve start", "name", name)
			err := srv.Serve(gctx)
			if err != nil && !m.cfg.IsNormalError(err) && gctx.Err() == nil {
				m.cfg.Logger("ERROR", "serve error", "name", name, "err", err)
				if m.cfg.Metrics != nil {
					m.cfg.Metrics.IncServeError(name)
				}
				return err
			}
			m.cfg.Logger("INFO", "serve stop", "name", name, "err", errString(err))
			return nil
		})
	}

	waitCh := make(chan error, 1)
	go func() { waitCh <- g.Wait() }()

	var groupDone bool
	var groupErr error

	select {
	case <-ctx.Done():
		m.cfg.Logger("INFO", "context done; starting graceful stop")
	case err := <-waitCh:
		groupDone, groupErr = true, err
		if err != nil && !m.cfg.IsNormalError(err) {
			m.cfg.Logger("WARN", "group finished with error; starting graceful stop", "err", err)
		} else {
			m.cfg.Logger("INFO", "group finished; starting graceful stop")
		}
	}

	m.Stop()

	if groupDone {
		if groupErr != nil && !m.cfg.IsNormalError(groupErr) {
			return groupErr
		}
		return nil
	}

	select {
	case err := <-waitCh:
		if err != nil && !m.cfg.IsNormalError(err) {
			return err
		}
		return nil
	case <-time.After(m.cfg.ShutdownTimeout + 2*time.Second):
		return fmt.Errorf("graceful: wait group timeout after %s", m.cfg.ShutdownTimeout)
	}
}

func (m *Manager) Stop() {
	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		return
	}
	m.stopped = true
	m.mu.Unlock()

	started := time.Now()
	var forcedAny atomic.Bool

	// Глобальный дедлайн
	globalCtx, globalCancel := context.WithTimeout(context.Background(), m.cfg.ShutdownTimeout)
	defer globalCancel()

	deadline, hasDeadline := globalCtx.Deadline()

	// Вместо sync.WaitGroup — errgroup
	g, _ := errgroup.WithContext(globalCtx)

	for _, s := range m.servers {
		srv := s
		g.Go(func() error {
			name := safeName(srv)

			// Локальный контекст «остатка времени» для сервера
			var srvCtx context.Context
			var cancel context.CancelFunc
			if hasDeadline {
				srvCtx, cancel = context.WithDeadline(context.Background(), deadline)
			} else {
				srvCtx, cancel = context.WithCancel(context.Background())
			}
			defer cancel()

			// Пытаемся graceful в рамках локального контекста
			if err := srv.GracefulStopWithTimeout(srvCtx); err != nil {
				m.cfg.Logger("WARN", "graceful stop error; forcing", "name", name, "err", err)
				srv.ForceStop()
				forcedAny.Store(true)
				if m.cfg.Metrics != nil {
					m.cfg.Metrics.IncServerStopResult(name, "force")
				}
				return nil
			}

			// Если дошли сюда — сервер сам закрылся в свой дедлайн: это success,
			// даже если глобальный уже истёк к этому моменту.
			m.cfg.Logger("INFO", "graceful stop done", "name", name)
			if m.cfg.Metrics != nil {
				m.cfg.Metrics.IncServerStopResult(name, "success")
			}
			return nil
		})
	}

	// Ждем завершения всех горутин
	_ = g.Wait()

	if m.cfg.Metrics != nil {
		m.cfg.Metrics.ObserveGracefulDuration(time.Since(started))
		result := "success"
		if forcedAny.Load() {
			result = "force"
		}
		m.cfg.Metrics.IncStopTotal(result)
	}
}

func DefaultIsNormalErr(err error) bool {
	if err == nil {
		return true
	}
	// gRPC: штатное завершение Serve после Stop/GracefulStop.
	if errors.Is(err, grpc.ErrServerStopped) {
		return true
	}

	// HTTP: штатное завершение.
	if errors.Is(err, http.ErrServerClosed) {
		return true
	}

	// Частые «нормальные» ошибки от сетевых серверов при закрытии листенера.
	msg := err.Error()
	if strings.Contains(msg, "use of closed network connection") {
		return true
	}
	// Иногда встречается как строка (на случай нестандартной обёртки).
	if strings.Contains(msg, "the server has been stopped") {
		return true
	}

	// gRPC TLS-handshake в момент остановки — тоже не считаем фаталом.
	if strings.Contains(msg, "Server.Serve failed to complete security handshake") {
		return true
	}
	return false
}

func errString(err error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Error()
}
func safeName(s Server) string {
	if s == nil {
		return "server"
	}
	if n := s.Name(); n != "" {
		return n
	}
	return "server"
}

//go:build unix

package shutdown

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"
)

func Test_Manager_HandleSignals_SIGTERM_StopsRun(t *testing.T) {
	t.Parallel()

	// Фейковый сервер, который ждёт ctx.Done()
	s := newFakeServer("waiter")
	s.waitForCtx = true

	m := New(Config{
		ShutdownTimeout: 300 * time.Millisecond,
		HandleSignals:   true,
	})
	m.Add(s)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- m.Run(ctx) }()

	// Даём время подписаться на сигналы и стартовать
	time.Sleep(50 * time.Millisecond)

	// Отправляем себе SIGTERM
	p, _ := os.FindProcess(os.Getpid())
	if err := p.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("send SIGTERM: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error after SIGTERM: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not stop after SIGTERM")
	}
}

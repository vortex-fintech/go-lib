//go:build integration

package shutdown

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/vortex-fintech/go-lib/runtime/shutdown/adapters"
	"google.golang.org/grpc"
)

func Test_Manager_With_GRPCAdapter_GracefulCancel_OK(t *testing.T) {
	t.Parallel()

	// Реальный listener на localhost:0
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer lis.Close()

	// Реальный gRPC server (без зарегистрированных сервисов, нам это не нужно)
	srv := grpc.NewServer()
	ad := &adapters.GRPC{Srv: srv, Lis: lis, NameStr: "grpc-int"}

	// Менеджер с небольшим таймаутом
	m := New(Config{
		ShutdownTimeout: 500 * time.Millisecond,
	})

	m.Add(ad)

	// Запускаем и через малое время отменяем контекст,
	// что должно привести к GracefulStop и штатному выходу Run(nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- m.Run(ctx) }()

	time.Sleep(80 * time.Millisecond) // даём серверу стартануть
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not finish after cancel")
	}
}

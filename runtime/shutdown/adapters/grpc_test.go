//go:build unit

package adapters

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
)

func TestGRPCAdapter_ServeAndGracefulStop(t *testing.T) {
	t.Parallel()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer lis.Close()

	s := grpc.NewServer()
	ad := &GRPC{Srv: s, Lis: lis, NameStr: "grpc-test"}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- ad.Serve(ctx) }()

	// give time to start
	time.Sleep(50 * time.Millisecond)

	// Graceful stop should complete quickly
	shCtx, shCancel := context.WithTimeout(context.Background(), time.Second)
	defer shCancel()
	if err := ad.GracefulStopWithTimeout(shCtx); err != nil {
		t.Fatalf("graceful stop: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not exit after GracefulStop")
	}
}

func TestGRPCAdapter_ForceStop(t *testing.T) {
	t.Parallel()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer lis.Close()

	s := grpc.NewServer()
	ad := &GRPC{Srv: s, Lis: lis}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- ad.Serve(ctx) }()

	time.Sleep(50 * time.Millisecond)
	ad.ForceStop()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not exit after ForceStop")
	}
}

func TestGRPCAdapter_DefaultName(t *testing.T) {
	t.Parallel()
	ad := &GRPC{Srv: grpc.NewServer()}
	if got := ad.Name(); got != "grpc" {
		t.Fatalf("expected default name 'grpc', got %q", got)
	}
}

func TestGRPCAdapter_CustomName(t *testing.T) {
	t.Parallel()
	ad := &GRPC{Srv: grpc.NewServer(), NameStr: "my-grpc"}
	if got := ad.Name(); got != "my-grpc" {
		t.Fatalf("expected custom name 'my-grpc', got %q", got)
	}
}

func TestGRPCAdapter_NilSrv_Serve(t *testing.T) {
	t.Parallel()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer lis.Close()

	ad := &GRPC{Srv: nil, Lis: lis}
	err = ad.Serve(context.Background())
	if err == nil {
		t.Fatal("expected error for nil Srv")
	}
}

func TestGRPCAdapter_NilLis_Serve(t *testing.T) {
	t.Parallel()
	ad := &GRPC{Srv: grpc.NewServer(), Lis: nil}
	err := ad.Serve(context.Background())
	if err == nil {
		t.Fatal("expected error for nil Lis")
	}
}

func TestGRPCAdapter_NilSrv_GracefulStop(t *testing.T) {
	t.Parallel()
	ad := &GRPC{Srv: nil}
	err := ad.GracefulStopWithTimeout(context.Background())
	if err == nil {
		t.Fatal("expected error for nil Srv")
	}
}

func TestGRPCAdapter_NilSrv_ForceStop(t *testing.T) {
	t.Parallel()
	ad := &GRPC{Srv: nil}
	ad.ForceStop() // should not panic
}

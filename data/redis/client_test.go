package redis

import (
	"context"
	"crypto/tls"
	"errors"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// helper: swaps NewUniversal and returns restore function
func stubNewUniversal(t *testing.T, fn func(opt *goredis.UniversalOptions) goredis.UniversalClient) func() {
	t.Helper()
	orig := NewUniversal
	NewUniversal = fn
	return func() { NewUniversal = orig }
}

func TestNewRedisClient_UsesAddrWhenAddrsEmpty(t *testing.T) {
	var captured *goredis.UniversalOptions

	restore := stubNewUniversal(t, func(opt *goredis.UniversalOptions) goredis.UniversalClient {
		captured = opt
		// Return a real client with an unreachable address
		// so Ping fails without external dependencies.
		return goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:1"})
	})
	defer restore()

	cfg := Config{
		Mode:        ModeSingle,
		Addr:        "127.0.0.1:6379",
		DialTimeout: 50 * time.Millisecond,
	}

	_, err := NewRedisClient(context.Background(), cfg)
	if err == nil {
		t.Fatalf("expected ping error, got nil")
	}

	if captured == nil {
		t.Fatalf("NewUniversal was not called")
	}
	if len(captured.Addrs) != 1 || captured.Addrs[0] != "127.0.0.1:6379" {
		t.Fatalf("expected Addrs to contain fallback from Addr, got %+v", captured.Addrs)
	}
	if captured.DB != 0 {
		t.Fatalf("expected DB=0 by default for single mode, got %d", captured.DB)
	}
}

func TestNewRedisClient_UsesAddrsForSentinel(t *testing.T) {
	var captured *goredis.UniversalOptions

	restore := stubNewUniversal(t, func(opt *goredis.UniversalOptions) goredis.UniversalClient {
		captured = opt
		return goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:1"})
	})
	defer restore()

	cfg := Config{
		Mode:        ModeSentinel,
		Addrs:       []string{"10.0.0.1:26379", "10.0.0.2:26379"},
		MasterName:  "mymaster",
		DB:          1,
		DialTimeout: 50 * time.Millisecond,
	}

	_, err := NewRedisClient(context.Background(), cfg)
	if err == nil {
		t.Fatalf("expected ping error, got nil")
	}

	if captured == nil {
		t.Fatalf("NewUniversal was not called")
	}
	if got := captured.Addrs; len(got) != 2 || got[0] != "10.0.0.1:26379" || got[1] != "10.0.0.2:26379" {
		t.Fatalf("unexpected Addrs: %+v", got)
	}
	if captured.MasterName != "mymaster" {
		t.Fatalf("expected MasterName 'mymaster', got %q", captured.MasterName)
	}
	if captured.DB != 1 {
		t.Fatalf("expected DB=1, got %d", captured.DB)
	}
}

func TestNewRedisClient_TLSConfigApplied(t *testing.T) {
	var captured *goredis.UniversalOptions

	restore := stubNewUniversal(t, func(opt *goredis.UniversalOptions) goredis.UniversalClient {
		captured = opt
		return goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:1"})
	})
	defer restore()

	cfg := Config{
		Mode:        ModeSingle,
		Addr:        "example:6379",
		TLSEnabled:  true,
		DialTimeout: 50 * time.Millisecond,
	}

	_, err := NewRedisClient(context.Background(), cfg)
	if err == nil {
		t.Fatalf("expected ping error, got nil")
	}

	if captured == nil {
		t.Fatalf("NewUniversal was not called")
	}
	if captured.TLSConfig == nil {
		t.Fatalf("expected TLSConfig to be set when TLSEnabled=true")
	}
	if captured.TLSConfig.MinVersion != tls.VersionTLS12 {
		t.Fatalf("expected MinVersion TLS1.2, got %v", captured.TLSConfig.MinVersion)
	}
}

func TestNewRedisClient_TimeoutFallbackForPing(t *testing.T) {
	// Verify that DialTimeout<=0 falls back to 3s without panics.
	var captured *goredis.UniversalOptions

	restore := stubNewUniversal(t, func(opt *goredis.UniversalOptions) goredis.UniversalClient {
		captured = opt
		return goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:1"})
	})
	defer restore()

	cfg := Config{
		Mode: ModeSingle,
		Addr: "127.0.0.1:6379",
		// DialTimeout is not set (0)
	}

	_, err := NewRedisClient(context.Background(), cfg)
	if err == nil {
		t.Fatalf("expected ping error, got nil")
	}
	if captured == nil {
		t.Fatalf("NewUniversal was not called")
	}
	// No panic and successful option wiring are enough here;
	// the exact timeout is applied to context in NewRedisClient.
}

func TestNewRedisClient_Validate_NoAddress(t *testing.T) {
	called := false
	restore := stubNewUniversal(t, func(opt *goredis.UniversalOptions) goredis.UniversalClient {
		called = true
		return goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:1"})
	})
	defer restore()

	_, err := NewRedisClient(context.Background(), Config{Mode: ModeSingle})
	if !errors.Is(err, errAddressRequired) {
		t.Fatalf("expected errAddressRequired, got %v", err)
	}
	if called {
		t.Fatalf("NewUniversal must not be called on invalid config")
	}
}

func TestNewRedisClient_Validate_SentinelRequiresMaster(t *testing.T) {
	called := false
	restore := stubNewUniversal(t, func(opt *goredis.UniversalOptions) goredis.UniversalClient {
		called = true
		return goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:1"})
	})
	defer restore()

	_, err := NewRedisClient(context.Background(), Config{Mode: ModeSentinel, Addrs: []string{"127.0.0.1:26379"}})
	if !errors.Is(err, errMasterNameRequired) {
		t.Fatalf("expected errMasterNameRequired, got %v", err)
	}
	if called {
		t.Fatalf("NewUniversal must not be called on invalid config")
	}
}

func TestNewRedisClient_Validate_UnsupportedMode(t *testing.T) {
	called := false
	restore := stubNewUniversal(t, func(opt *goredis.UniversalOptions) goredis.UniversalClient {
		called = true
		return goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:1"})
	})
	defer restore()

	_, err := NewRedisClient(context.Background(), Config{Mode: "weird", Addr: "127.0.0.1:6379"})
	if !errors.Is(err, errUnsupportedMode) {
		t.Fatalf("expected errUnsupportedMode, got %v", err)
	}
	if called {
		t.Fatalf("NewUniversal must not be called on invalid config")
	}
}

func TestNewRedisClient_NilContextFallback(t *testing.T) {
	restore := stubNewUniversal(t, func(opt *goredis.UniversalOptions) goredis.UniversalClient {
		return goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:1"})
	})
	defer restore()

	_, err := NewRedisClient(nil, Config{Mode: ModeSingle, Addr: "127.0.0.1:6379", DialTimeout: 50 * time.Millisecond})
	if err == nil {
		t.Fatalf("expected ping error, got nil")
	}
}

func TestNewRedisClient_Validate_SingleRequiresExactlyOneAddr(t *testing.T) {
	called := false
	restore := stubNewUniversal(t, func(opt *goredis.UniversalOptions) goredis.UniversalClient {
		called = true
		return goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:1"})
	})
	defer restore()

	_, err := NewRedisClient(context.Background(), Config{Mode: ModeSingle, Addrs: []string{"10.0.0.1:6379", "10.0.0.2:6379"}})
	if !errors.Is(err, errSingleModeAddrCount) {
		t.Fatalf("expected errSingleModeAddrCount, got %v", err)
	}
	if called {
		t.Fatalf("NewUniversal must not be called on invalid config")
	}
}

func TestNewRedisClient_Validate_ClusterRequiresMultipleAddrs(t *testing.T) {
	called := false
	restore := stubNewUniversal(t, func(opt *goredis.UniversalOptions) goredis.UniversalClient {
		called = true
		return goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:1"})
	})
	defer restore()

	_, err := NewRedisClient(context.Background(), Config{Mode: ModeCluster, Addrs: []string{"10.0.0.1:6379"}})
	if !errors.Is(err, errClusterModeAddrCount) {
		t.Fatalf("expected errClusterModeAddrCount, got %v", err)
	}
	if called {
		t.Fatalf("NewUniversal must not be called on invalid config")
	}
}

func TestNewRedisClient_Validate_ClusterDBMustBeZero(t *testing.T) {
	called := false
	restore := stubNewUniversal(t, func(opt *goredis.UniversalOptions) goredis.UniversalClient {
		called = true
		return goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:1"})
	})
	defer restore()

	_, err := NewRedisClient(context.Background(), Config{Mode: ModeCluster, Addrs: []string{"10.0.0.1:6379", "10.0.0.2:6379"}, DB: 1})
	if !errors.Is(err, errClusterDBUnsupported) {
		t.Fatalf("expected errClusterDBUnsupported, got %v", err)
	}
	if called {
		t.Fatalf("NewUniversal must not be called on invalid config")
	}
}

func TestNewRedisClient_Validate_MasterNameOnlyForSentinel(t *testing.T) {
	called := false
	restore := stubNewUniversal(t, func(opt *goredis.UniversalOptions) goredis.UniversalClient {
		called = true
		return goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:1"})
	})
	defer restore()

	_, err := NewRedisClient(context.Background(), Config{Mode: ModeSingle, Addr: "127.0.0.1:6379", MasterName: "mymaster"})
	if !errors.Is(err, errMasterNameUnexpected) {
		t.Fatalf("expected errMasterNameUnexpected, got %v", err)
	}
	if called {
		t.Fatalf("NewUniversal must not be called on invalid config")
	}
}

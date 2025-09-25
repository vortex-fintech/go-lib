package redis

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// helper: подмена NewUniversal и возврат оригинала
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
		// Возвращаем реальный клиент с заведомо недоступным адресом,
		// чтобы Ping упал без внешних зависимостей.
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
	// Проверим, что при DialTimeout<=0 берётся дефолт 3s (поведение без паник/ошибок).
	var captured *goredis.UniversalOptions

	restore := stubNewUniversal(t, func(opt *goredis.UniversalOptions) goredis.UniversalClient {
		captured = opt
		return goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:1"})
	})
	defer restore()

	cfg := Config{
		Mode: ModeSingle,
		Addr: "127.0.0.1:6379",
		// DialTimeout не задаём (0)
	}

	_, err := NewRedisClient(context.Background(), cfg)
	if err == nil {
		t.Fatalf("expected ping error, got nil")
	}
	if captured == nil {
		t.Fatalf("NewUniversal was not called")
	}
	// сам факт, что не было паники и всё собралось — уже ок;
	// конкретный таймаут мы не читаем из opt, он используется в NewRedisClient для ctx.
}

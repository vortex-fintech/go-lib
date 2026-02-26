package replay

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestInMemoryChecker_SubSecondTTL(t *testing.T) {
	t.Parallel()

	c := NewInMemoryChecker(MemoryOptions{TTL: 500 * time.Millisecond})

	seen, err := c.SeenJTI(context.Background(), "wallet", "jti-1", 500*time.Millisecond)
	if err != nil {
		t.Fatalf("first SeenJTI error: %v", err)
	}
	if seen {
		t.Fatalf("first SeenJTI should not be replay")
	}

	seen, err = c.SeenJTI(context.Background(), "wallet", "jti-1", 500*time.Millisecond)
	if err != nil {
		t.Fatalf("second SeenJTI error: %v", err)
	}
	if !seen {
		t.Fatalf("second SeenJTI should be replay")
	}

	time.Sleep(700 * time.Millisecond)

	seen, err = c.SeenJTI(context.Background(), "wallet", "jti-1", 500*time.Millisecond)
	if err != nil {
		t.Fatalf("third SeenJTI error: %v", err)
	}
	if seen {
		t.Fatalf("entry should expire after sub-second TTL")
	}
}

func TestInMemoryChecker_UsesDefaultTTLWhenNonPositive(t *testing.T) {
	t.Parallel()

	c := NewInMemoryChecker(MemoryOptions{TTL: 200 * time.Millisecond})

	seen, err := c.SeenJTI(context.Background(), "wallet", "jti-2", 0)
	if err != nil {
		t.Fatalf("first SeenJTI error: %v", err)
	}
	if seen {
		t.Fatalf("first SeenJTI should not be replay")
	}

	seen, err = c.SeenJTI(context.Background(), "wallet", "jti-2", -time.Second)
	if err != nil {
		t.Fatalf("second SeenJTI error: %v", err)
	}
	if !seen {
		t.Fatalf("second SeenJTI should be replay when default TTL is active")
	}
}

func TestInMemoryChecker_InvalidEffectiveTTL(t *testing.T) {
	t.Parallel()

	c := NewInMemoryChecker(MemoryOptions{TTL: 0})

	seen, err := c.SeenJTI(context.Background(), "wallet", "jti-ttl-0", 0)
	if !seen {
		t.Fatalf("expected fail-closed seen=true when ttl is invalid")
	}
	if !errors.Is(err, ErrInvalidTTL) {
		t.Fatalf("expected ErrInvalidTTL, got %v", err)
	}
}

func TestInMemoryChecker_MaxItems(t *testing.T) {
	t.Parallel()

	c := NewInMemoryChecker(MemoryOptions{
		TTL:      time.Hour,
		MaxItems: 3,
	})

	seen, _ := c.SeenJTI(context.Background(), "ns", "jti-1", time.Hour)
	if seen {
		t.Fatal("jti-1 should not be replay")
	}
	seen, _ = c.SeenJTI(context.Background(), "ns", "jti-2", time.Hour)
	if seen {
		t.Fatal("jti-2 should not be replay")
	}
	seen, _ = c.SeenJTI(context.Background(), "ns", "jti-3", time.Hour)
	if seen {
		t.Fatal("jti-3 should not be replay")
	}

	seen, _ = c.SeenJTI(context.Background(), "ns", "jti-4", time.Hour)
	if seen {
		t.Fatal("jti-4 should not be replay (evicted oldest)")
	}

	c.mu.Lock()
	if len(c.items) > 3 {
		t.Fatalf("items count should be <= MaxItems, got %d", len(c.items))
	}
	c.mu.Unlock()
}

func TestInMemoryChecker_Namespaces(t *testing.T) {
	t.Parallel()

	c := NewInMemoryChecker(MemoryOptions{TTL: time.Hour})

	seen, _ := c.SeenJTI(context.Background(), "ns-a", "jti-1", time.Hour)
	if seen {
		t.Fatal("ns-a:jti-1 should not be replay")
	}

	seen, _ = c.SeenJTI(context.Background(), "ns-b", "jti-1", time.Hour)
	if seen {
		t.Fatal("ns-b:jti-1 should not be replay (different namespace)")
	}

	seen, _ = c.SeenJTI(context.Background(), "ns-a", "jti-1", time.Hour)
	if !seen {
		t.Fatal("ns-a:jti-1 should be replay")
	}
}

func TestInMemoryChecker_AsAuthzCallback(t *testing.T) {
	t.Parallel()

	c := NewInMemoryChecker(MemoryOptions{TTL: time.Hour})
	cb := c.AsAuthzCallback("wallet", time.Hour)

	if cb("jti-1") {
		t.Fatal("first call should return false")
	}
	if !cb("jti-1") {
		t.Fatal("second call should return true (replay)")
	}
	if cb("jti-2") {
		t.Fatal("different jti should return false")
	}
}

func TestRedisChecker_InvalidTTL_FailClosed(t *testing.T) {
	t.Parallel()

	c := NewRedisChecker(nil, RedisOptions{FailOpen: false})

	seen, err := c.SeenJTI(context.Background(), "wallet", "jti-1", 0)
	if !seen {
		t.Fatal("expected seen=true in fail-closed mode")
	}
	if !errors.Is(err, ErrInvalidTTL) {
		t.Fatalf("expected ErrInvalidTTL, got %v", err)
	}
}

func TestRedisChecker_InvalidTTL_FailOpen(t *testing.T) {
	t.Parallel()

	c := NewRedisChecker(nil, RedisOptions{FailOpen: true})

	seen, err := c.SeenJTI(context.Background(), "wallet", "jti-1", -time.Second)
	if seen {
		t.Fatal("expected seen=false in fail-open mode")
	}
	if err != nil {
		t.Fatalf("expected nil error in fail-open mode, got %v", err)
	}
}

func TestRedisChecker_NilClient_FailClosed(t *testing.T) {
	t.Parallel()

	c := NewRedisChecker(nil, RedisOptions{FailOpen: false})

	seen, err := c.SeenJTI(context.Background(), "wallet", "jti-1", time.Minute)
	if !seen {
		t.Fatal("expected seen=true for nil client in fail-closed mode")
	}
	if !errors.Is(err, ErrNilRedisClient) {
		t.Fatalf("expected ErrNilRedisClient, got %v", err)
	}
}

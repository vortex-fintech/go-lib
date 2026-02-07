package replay

import (
	"context"
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

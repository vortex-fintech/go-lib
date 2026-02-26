package domain_test

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vortex-fintech/go-lib/foundation/domain"
)

type testEvent struct {
	name string
	at   time.Time
	id   uuid.UUID
	ver  int32
}

func (e testEvent) EventName() string     { return e.name }
func (e testEvent) OccurredAt() time.Time { return e.at }
func (e testEvent) EventID() uuid.UUID    { return e.id }
func (e testEvent) SchemaVer() int32      { return e.ver }

func TestEventBuffer_RecordPeekPull(t *testing.T) {
	var b domain.EventBuffer

	if b.Len() != 0 {
		t.Fatalf("expected empty")
	}

	// nil must not be recorded.
	b.Record(nil)
	if b.Len() != 0 {
		t.Fatalf("expected still empty")
	}

	e1 := testEvent{name: "e1", at: time.Now().UTC(), id: uuid.New(), ver: 1}
	e2 := testEvent{name: "e2", at: time.Now().UTC(), id: uuid.New(), ver: 1}

	b.Record(e1)
	b.Record(e2)

	if b.Len() != 2 {
		t.Fatalf("expected len=2, got %d", b.Len())
	}

	peek := b.Peek()
	if len(peek) != 2 {
		t.Fatalf("expected peek len=2, got %d", len(peek))
	}

	// Peek must return a copy: mutating snapshot must not affect the buffer.
	peek[0] = testEvent{name: "hijack", at: time.Now().UTC(), id: uuid.New(), ver: 1}
	peek2 := b.Peek()
	if peek2[0].EventName() != "e1" {
		t.Fatalf("expected buffer unchanged, got %q", peek2[0].EventName())
	}

	out := b.Pull()
	if len(out) != 2 {
		t.Fatalf("expected pulled len=2, got %d", len(out))
	}
	if b.Len() != 0 {
		t.Fatalf("expected buffer cleared after pull")
	}
	if b.Peek() != nil && len(b.Peek()) != 0 {
		t.Fatalf("expected empty peek after pull")
	}
}

func TestEventBuffer_Clear(t *testing.T) {
	var b domain.EventBuffer

	e := testEvent{name: "e", at: time.Now().UTC(), id: uuid.New(), ver: 1}
	b.Record(e)
	if b.Len() != 1 {
		t.Fatalf("expected len=1, got %d", b.Len())
	}

	b.Clear()
	if b.Len() != 0 {
		t.Fatalf("expected cleared")
	}
}

func TestEventBuffer_ConcurrentRecordAndLen(t *testing.T) {
	var b domain.EventBuffer

	const workers = 8
	const perWorker = 50

	var wg sync.WaitGroup
	wg.Add(workers)

	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				b.Record(testEvent{name: "e", at: time.Now().UTC(), id: uuid.New(), ver: 1})
				_ = b.Len()
			}
		}()
	}

	wg.Wait()

	if b.Len() != workers*perWorker {
		t.Fatalf("expected len=%d, got %d", workers*perWorker, b.Len())
	}
}

func TestEventBuffer_Record_TypedNilPointerIgnored(t *testing.T) {
	var b domain.EventBuffer
	var e *testEvent

	b.Record(e)
	if b.Len() != 0 {
		t.Fatalf("expected empty buffer for typed nil event")
	}
}

func TestEventBuffer_RecordStrict(t *testing.T) {
	t.Run("rejects nil", func(t *testing.T) {
		var b domain.EventBuffer
		err := b.RecordStrict(nil)
		if err == nil {
			t.Fatalf("expected error")
		}
		if !errors.Is(err, domain.ErrInvalidEvent) {
			t.Fatalf("expected ErrInvalidEvent, got %v", err)
		}
	})

	t.Run("rejects non validatable events", func(t *testing.T) {
		var b domain.EventBuffer
		err := b.RecordStrict(testEvent{name: "e", at: time.Now().UTC(), id: uuid.New(), ver: 1})
		if err == nil {
			t.Fatalf("expected error")
		}
		if !errors.Is(err, domain.ErrInvalidEvent) {
			t.Fatalf("expected ErrInvalidEvent, got %v", err)
		}
		if b.Len() != 0 {
			t.Fatalf("expected no recorded events")
		}
	})

	t.Run("records valid events", func(t *testing.T) {
		var b domain.EventBuffer
		e := domain.BaseEvent{
			Name:          "user.created",
			At:            time.Now().UTC(),
			ID:            uuid.New(),
			SchemaVersion: 1,
			Producer:      "user-service",
		}

		if err := b.RecordStrict(e); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.Len() != 1 {
			t.Fatalf("expected len=1, got %d", b.Len())
		}
	})

	t.Run("rejects invalid events", func(t *testing.T) {
		var b domain.EventBuffer
		e := domain.BaseEvent{
			Name:          "user.created",
			At:            time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("UTC0", 0)),
			ID:            uuid.New(),
			SchemaVersion: 1,
			Producer:      "user-service",
		}

		err := b.RecordStrict(e)
		if err == nil {
			t.Fatalf("expected error")
		}
		if !errors.Is(err, domain.ErrInvalidEvent) {
			t.Fatalf("expected ErrInvalidEvent, got %v", err)
		}
		if b.Len() != 0 {
			t.Fatalf("expected len=0, got %d", b.Len())
		}
	})

	t.Run("rejects events exceeding default limits", func(t *testing.T) {
		var b domain.EventBuffer
		e := domain.BaseEvent{
			Name:          strings.Repeat("n", domain.DefaultEventLimits.MaxNameRunes+1),
			At:            time.Now().UTC(),
			ID:            uuid.New(),
			SchemaVersion: 1,
			Producer:      "user-service",
		}

		err := b.RecordStrict(e)
		if !errors.Is(err, domain.ErrInvalidEventNameTooLong) {
			t.Fatalf("expected ErrInvalidEventNameTooLong, got %v", err)
		}
		if b.Len() != 0 {
			t.Fatalf("expected len=0, got %d", b.Len())
		}
	})
}

package domain_test

import (
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

	// nil не должен записываться
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

	// Peek должен возвращать копию slice: меняем peek — не должно влиять на буфер
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

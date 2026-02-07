package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vortex-fintech/go-lib/foundation/domain"
	"github.com/vortex-fintech/go-lib/foundation/timeutil"
)

func TestNewBaseEvent_UsesTimeutilClockAndUTC(t *testing.T) {
	restore := timeutil.WithDefault(timeutil.NewFrozenClock(time.Date(2025, 12, 13, 1, 2, 3, 0, time.UTC)))
	t.Cleanup(restore)

	e, err := domain.NewBaseEvent("pii.address.created", "pii-service")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantAt := time.Date(2025, 12, 13, 1, 2, 3, 0, time.UTC)
	if !e.OccurredAt().Equal(wantAt) {
		t.Fatalf("want At=%v, got %v", wantAt, e.OccurredAt())
	}
	if e.OccurredAt().Location() != time.UTC {
		t.Fatalf("expected UTC location, got %v", e.OccurredAt().Location())
	}
	if e.EventID() == uuid.Nil {
		t.Fatalf("expected non-nil ID")
	}
	if e.SchemaVer() != 1 {
		t.Fatalf("expected schema=1, got %d", e.SchemaVer())
	}
}

func TestBaseEvent_Validate_OK(t *testing.T) {
	e := domain.BaseEvent{
		Name:          "x",
		At:            time.Date(2025, 12, 13, 0, 0, 0, 0, time.UTC),
		ID:            uuid.New(),
		SchemaVersion: 1,
		Producer:      "svc",
	}
	if err := e.Validate(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestBaseEvent_Validate_ErrorsAreIsable(t *testing.T) {
	cases := []struct {
		name     string
		ev       domain.BaseEvent
		wantIs   error
		wantWrap error
	}{
		{
			name: "invalid name",
			ev: domain.BaseEvent{
				Name:          " ",
				At:            time.Date(2025, 12, 13, 0, 0, 0, 0, time.UTC),
				ID:            uuid.New(),
				SchemaVersion: 1,
				Producer:      "svc",
			},
			wantIs:   domain.ErrInvalidEvent,
			wantWrap: domain.ErrInvalidEventName,
		},
		{
			name: "invalid producer",
			ev: domain.BaseEvent{
				Name:          "x",
				At:            time.Date(2025, 12, 13, 0, 0, 0, 0, time.UTC),
				ID:            uuid.New(),
				SchemaVersion: 1,
				Producer:      " ",
			},
			wantIs:   domain.ErrInvalidEvent,
			wantWrap: domain.ErrInvalidEventProducer,
		},
		{
			name: "invalid time not utc location",
			ev: domain.BaseEvent{
				Name:          "x",
				At:            time.Date(2025, 12, 13, 0, 0, 0, 0, time.FixedZone("UTC0", 0)), // offset 0, но НЕ time.UTC
				ID:            uuid.New(),
				SchemaVersion: 1,
				Producer:      "svc",
			},
			wantIs:   domain.ErrInvalidEvent,
			wantWrap: domain.ErrInvalidEventTime,
		},
		{
			name: "invalid id",
			ev: domain.BaseEvent{
				Name:          "x",
				At:            time.Date(2025, 12, 13, 0, 0, 0, 0, time.UTC),
				ID:            uuid.Nil,
				SchemaVersion: 1,
				Producer:      "svc",
			},
			wantIs:   domain.ErrInvalidEvent,
			wantWrap: domain.ErrInvalidEventID,
		},
		{
			name: "invalid schema",
			ev: domain.BaseEvent{
				Name:          "x",
				At:            time.Date(2025, 12, 13, 0, 0, 0, 0, time.UTC),
				ID:            uuid.New(),
				SchemaVersion: 0,
				Producer:      "svc",
			},
			wantIs:   domain.ErrInvalidEvent,
			wantWrap: domain.ErrInvalidEventSchema,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.ev.Validate()
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !errors.Is(err, tc.wantIs) {
				t.Fatalf("expected errors.Is(err, %v) == true; err=%v", tc.wantIs, err)
			}
			if !errors.Is(err, tc.wantWrap) {
				t.Fatalf("expected errors.Is(err, %v) == true; err=%v", tc.wantWrap, err)
			}
		})
	}
}

func TestBaseEvent_WithSchema_IncreasesOnly(t *testing.T) {
	e := domain.BaseEvent{SchemaVersion: 2}

	e2 := e.WithSchema(0)
	if e2.SchemaVersion != 2 {
		t.Fatalf("expected unchanged schema, got %d", e2.SchemaVersion)
	}

	e3 := e.WithSchema(1)
	if e3.SchemaVersion != 2 {
		t.Fatalf("expected unchanged schema, got %d", e3.SchemaVersion)
	}

	e4 := e.WithSchema(3)
	if e4.SchemaVersion != 3 {
		t.Fatalf("expected schema=3, got %d", e4.SchemaVersion)
	}
}

func TestBaseEvent_WithMeta_CopyOnWrite(t *testing.T) {
	e1 := domain.BaseEvent{
		Name:          "x",
		At:            time.Date(2025, 12, 13, 0, 0, 0, 0, time.UTC),
		ID:            uuid.New(),
		SchemaVersion: 1,
		Producer:      "svc",
		Meta:          map[string]string{"a": "1"},
	}

	e2 := e1.WithMeta("b", "2")

	// меняем исходную карту
	e1.Meta["a"] = "changed"
	if e2.Meta["a"] != "1" {
		t.Fatalf("expected e2 meta unaffected, got %q", e2.Meta["a"])
	}
	if e2.Meta["b"] != "2" {
		t.Fatalf("expected e2 meta b=2, got %q", e2.Meta["b"])
	}
}

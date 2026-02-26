package domain_test

import (
	"errors"
	"strings"
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
				At:            time.Date(2025, 12, 13, 0, 0, 0, 0, time.FixedZone("UTC0", 0)), // offset 0, but not time.UTC
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

	// Mutate the source map.
	e1.Meta["a"] = "changed"
	if e2.Meta["a"] != "1" {
		t.Fatalf("expected e2 meta unaffected, got %q", e2.Meta["a"])
	}
	if e2.Meta["b"] != "2" {
		t.Fatalf("expected e2 meta b=2, got %q", e2.Meta["b"])
	}
}

func TestMustBaseEvent(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		e := domain.MustBaseEvent("user.created", "user-service")
		if err := e.Validate(); err != nil {
			t.Fatalf("expected valid event, got %v", err)
		}
	})

	t.Run("panic on invalid", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic")
			}
		}()

		_ = domain.MustBaseEvent(" ", "user-service")
	})
}

func TestBaseEvent_WithTrace_Trims(t *testing.T) {
	e := domain.BaseEvent{}
	out := e.WithTrace("  trace-1  ", "  corr-1  ")

	if out.TraceID != "trace-1" {
		t.Fatalf("expected trimmed trace id, got %q", out.TraceID)
	}
	if out.CorrelationID != "corr-1" {
		t.Fatalf("expected trimmed correlation id, got %q", out.CorrelationID)
	}
}

func TestBaseEvent_WithMeta_EmptyKeyIgnored(t *testing.T) {
	e := domain.BaseEvent{Meta: map[string]string{"a": "1"}}
	out := e.WithMeta("   ", "2")

	if len(out.Meta) != 1 {
		t.Fatalf("expected unchanged meta size, got %d", len(out.Meta))
	}
	if out.Meta["a"] != "1" {
		t.Fatalf("expected unchanged value, got %q", out.Meta["a"])
	}
}

func TestBaseEvent_WithMeta_NoAliasOnNoopUpdate(t *testing.T) {
	e1 := domain.BaseEvent{Meta: map[string]string{"a": "1"}}
	e2 := e1.WithMeta("a", "1")

	e1.Meta["a"] = "mutated"
	if e2.Meta["a"] != "1" {
		t.Fatalf("expected no aliasing on noop update, got %q", e2.Meta["a"])
	}
}

func TestBaseEvent_ValidateWithLimits(t *testing.T) {
	valid := domain.BaseEvent{
		Name:          "user.created",
		At:            time.Now().UTC(),
		ID:            uuid.New(),
		SchemaVersion: 1,
		Producer:      "user-service",
		Meta:          map[string]string{"k": "v"},
	}

	if err := valid.ValidateWithLimits(domain.EventLimits{}); err != nil {
		t.Fatalf("expected defaults to allow valid event, got %v", err)
	}

	limits := domain.EventLimits{
		MaxNameRunes:      32,
		MaxProducerRunes:  16,
		MaxMetaEntries:    2,
		MaxMetaKeyRunes:   4,
		MaxMetaValueRunes: 5,
	}

	t.Run("name too long", func(t *testing.T) {
		e := valid
		e.Name = strings.Repeat("n", 33)
		err := e.ValidateWithLimits(limits)
		if !errors.Is(err, domain.ErrInvalidEventNameTooLong) {
			t.Fatalf("expected ErrInvalidEventNameTooLong, got %v", err)
		}
	})

	t.Run("producer too long", func(t *testing.T) {
		e := valid
		e.Producer = strings.Repeat("p", 17)
		err := e.ValidateWithLimits(limits)
		if !errors.Is(err, domain.ErrInvalidEventProducerTooLong) {
			t.Fatalf("expected ErrInvalidEventProducerTooLong, got %v", err)
		}
	})

	t.Run("meta too many entries", func(t *testing.T) {
		e := valid
		e.Meta = map[string]string{"a": "1", "b": "2", "c": "3"}
		err := e.ValidateWithLimits(limits)
		if !errors.Is(err, domain.ErrInvalidEventMetaTooMany) {
			t.Fatalf("expected ErrInvalidEventMetaTooMany, got %v", err)
		}
	})

	t.Run("meta empty key", func(t *testing.T) {
		e := valid
		e.Meta = map[string]string{" ": "x"}
		err := e.ValidateWithLimits(limits)
		if !errors.Is(err, domain.ErrInvalidEventMetaKey) {
			t.Fatalf("expected ErrInvalidEventMetaKey, got %v", err)
		}
	})

	t.Run("meta key too long", func(t *testing.T) {
		e := valid
		e.Meta = map[string]string{"verylong": "x"}
		err := e.ValidateWithLimits(limits)
		if !errors.Is(err, domain.ErrInvalidEventMetaKeyTooLong) {
			t.Fatalf("expected ErrInvalidEventMetaKeyTooLong, got %v", err)
		}
	})

	t.Run("meta value too long", func(t *testing.T) {
		e := valid
		e.Meta = map[string]string{"ok": "value-too-long"}
		err := e.ValidateWithLimits(limits)
		if !errors.Is(err, domain.ErrInvalidEventMetaValueTooLong) {
			t.Fatalf("expected ErrInvalidEventMetaValueTooLong, got %v", err)
		}
	})
}

func TestBaseEvent_ValidateWithLimits_BoundaryAllowed(t *testing.T) {
	e := domain.BaseEvent{
		Name:          strings.Repeat("n", 10),
		At:            time.Now().UTC(),
		ID:            uuid.New(),
		SchemaVersion: 1,
		Producer:      strings.Repeat("p", 8),
		Meta:          map[string]string{"kkkk": "vvvvv"},
	}

	limits := domain.EventLimits{
		MaxNameRunes:      10,
		MaxProducerRunes:  8,
		MaxMetaEntries:    1,
		MaxMetaKeyRunes:   4,
		MaxMetaValueRunes: 5,
	}

	if err := e.ValidateWithLimits(limits); err != nil {
		t.Fatalf("expected boundary values to pass, got %v", err)
	}
}

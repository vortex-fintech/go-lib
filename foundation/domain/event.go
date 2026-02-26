package domain

import (
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/vortex-fintech/go-lib/foundation/timeutil"
)

type Event interface {
	EventName() string
	OccurredAt() time.Time
	EventID() uuid.UUID
	SchemaVer() int32
}

// Sentinel error for errors.Is checks.
var ErrInvalidEvent = errors.New("invalid event")

// Detailed reasons for logs and diagnostics.
var (
	ErrInvalidEventName     = errors.New("invalid event name")
	ErrInvalidEventProducer = errors.New("invalid event producer")
	ErrInvalidEventTime     = errors.New("invalid event time")
	ErrInvalidEventID       = errors.New("invalid event id")
	ErrInvalidEventSchema   = errors.New("invalid event schema version")
	ErrInvalidEventNil      = errors.New("nil event")

	ErrInvalidEventNameTooLong      = errors.New("event name too long")
	ErrInvalidEventProducerTooLong  = errors.New("event producer too long")
	ErrInvalidEventMetaTooMany      = errors.New("event meta has too many entries")
	ErrInvalidEventMetaKey          = errors.New("invalid event meta key")
	ErrInvalidEventMetaKeyTooLong   = errors.New("event meta key too long")
	ErrInvalidEventMetaValueTooLong = errors.New("event meta value too long")
)

type EventLimits struct {
	MaxNameRunes      int
	MaxProducerRunes  int
	MaxMetaEntries    int
	MaxMetaKeyRunes   int
	MaxMetaValueRunes int
}

var DefaultEventLimits = EventLimits{
	MaxNameRunes:      128,
	MaxProducerRunes:  64,
	MaxMetaEntries:    32,
	MaxMetaKeyRunes:   64,
	MaxMetaValueRunes: 256,
}

func (l EventLimits) normalized() EventLimits {
	out := l
	if out.MaxNameRunes <= 0 {
		out.MaxNameRunes = DefaultEventLimits.MaxNameRunes
	}
	if out.MaxProducerRunes <= 0 {
		out.MaxProducerRunes = DefaultEventLimits.MaxProducerRunes
	}
	if out.MaxMetaEntries <= 0 {
		out.MaxMetaEntries = DefaultEventLimits.MaxMetaEntries
	}
	if out.MaxMetaKeyRunes <= 0 {
		out.MaxMetaKeyRunes = DefaultEventLimits.MaxMetaKeyRunes
	}
	if out.MaxMetaValueRunes <= 0 {
		out.MaxMetaValueRunes = DefaultEventLimits.MaxMetaValueRunes
	}
	return out
}

// BaseEvent contains common event metadata.
// It is transport-agnostic and does not contain business payload.
type BaseEvent struct {
	// Core
	Name string
	At   time.Time

	// Idempotency / tracing.
	ID            uuid.UUID
	TraceID       string
	CorrelationID string
	CausationID   uuid.UUID

	// Compatibility.
	SchemaVersion int32

	// Producer metadata.
	Producer string

	// Extensible, non-PII metadata.
	Meta map[string]string
}

var _ Event = BaseEvent{} // compile-time contract

// NewBaseEvent creates a safe baseline event (UTC + UUID + schema v1).
func NewBaseEvent(name, producer string) (BaseEvent, error) {
	name = strings.TrimSpace(name)
	producer = strings.TrimSpace(producer)

	if name == "" {
		return BaseEvent{}, fmt.Errorf("%w: %w", ErrInvalidEvent, ErrInvalidEventName)
	}
	if producer == "" {
		return BaseEvent{}, fmt.Errorf("%w: %w", ErrInvalidEvent, ErrInvalidEventProducer)
	}

	return BaseEvent{
		Name:          name,
		At:            timeutil.Now().UTC(), // strict UTC
		ID:            uuid.New(),
		SchemaVersion: 1,
		Producer:      producer,
	}, nil
}

// MustBaseEvent panics on constructor error.
func MustBaseEvent(name, producer string) BaseEvent {
	e, err := NewBaseEvent(name, producer)
	if err != nil {
		panic(err)
	}
	return e
}

// WithTrace sets trace/correlation ids (usually from request context).
func (e BaseEvent) WithTrace(traceID, correlationID string) BaseEvent {
	e.TraceID = strings.TrimSpace(traceID)
	e.CorrelationID = strings.TrimSpace(correlationID)
	return e
}

// WithCausation sets causation id (for example parent EventID or CommandID).
func (e BaseEvent) WithCausation(id uuid.UUID) BaseEvent {
	e.CausationID = id
	return e
}

// WithMeta uses copy-on-write to avoid hidden map sharing.
func (e BaseEvent) WithMeta(k, v string) BaseEvent {
	k = strings.TrimSpace(k)
	if k == "" {
		return e
	}
	v = strings.TrimSpace(v)

	if e.Meta == nil {
		e.Meta = map[string]string{k: v}
		return e
	}

	m := make(map[string]string, len(e.Meta)+1)
	maps.Copy(m, e.Meta)
	m[k] = v
	e.Meta = m
	return e
}

func (e BaseEvent) WithSchema(ver int32) BaseEvent {
	if ver <= 0 {
		return e
	}
	if ver > e.SchemaVersion {
		e.SchemaVersion = ver
	}
	return e
}

// Validate performs strict event invariant checks.
// It returns ErrInvalidEvent with a wrapped specific reason.
func (e BaseEvent) Validate() error {
	if strings.TrimSpace(e.Name) == "" {
		return fmt.Errorf("%w: %w", ErrInvalidEvent, ErrInvalidEventName)
	}
	if strings.TrimSpace(e.Producer) == "" {
		return fmt.Errorf("%w: %w", ErrInvalidEvent, ErrInvalidEventProducer)
	}

	// At must be present and use time.UTC location (strict UTC contract).
	if e.At.IsZero() || e.At.Location() != time.UTC {
		return fmt.Errorf("%w: %w", ErrInvalidEvent, ErrInvalidEventTime)
	}

	if e.ID == uuid.Nil {
		return fmt.Errorf("%w: %w", ErrInvalidEvent, ErrInvalidEventID)
	}
	if e.SchemaVersion <= 0 {
		return fmt.Errorf("%w: %w", ErrInvalidEvent, ErrInvalidEventSchema)
	}
	return nil
}

func (e BaseEvent) ValidateWithLimits(limits EventLimits) error {
	if err := e.Validate(); err != nil {
		return err
	}

	l := limits.normalized()
	if utf8.RuneCountInString(strings.TrimSpace(e.Name)) > l.MaxNameRunes {
		return fmt.Errorf("%w: %w", ErrInvalidEvent, ErrInvalidEventNameTooLong)
	}
	if utf8.RuneCountInString(strings.TrimSpace(e.Producer)) > l.MaxProducerRunes {
		return fmt.Errorf("%w: %w", ErrInvalidEvent, ErrInvalidEventProducerTooLong)
	}
	if len(e.Meta) > l.MaxMetaEntries {
		return fmt.Errorf("%w: %w", ErrInvalidEvent, ErrInvalidEventMetaTooMany)
	}

	for k, v := range e.Meta {
		if strings.TrimSpace(k) == "" {
			return fmt.Errorf("%w: %w", ErrInvalidEvent, ErrInvalidEventMetaKey)
		}
		if utf8.RuneCountInString(k) > l.MaxMetaKeyRunes {
			return fmt.Errorf("%w: %w", ErrInvalidEvent, ErrInvalidEventMetaKeyTooLong)
		}
		if utf8.RuneCountInString(v) > l.MaxMetaValueRunes {
			return fmt.Errorf("%w: %w", ErrInvalidEvent, ErrInvalidEventMetaValueTooLong)
		}
	}

	return nil
}

// Interface implementation
func (e BaseEvent) EventName() string     { return e.Name }
func (e BaseEvent) OccurredAt() time.Time { return e.At } // UTC by contract
func (e BaseEvent) EventID() uuid.UUID    { return e.ID }
func (e BaseEvent) SchemaVer() int32      { return e.SchemaVersion }

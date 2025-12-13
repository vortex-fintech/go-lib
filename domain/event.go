package domain

import (
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Event interface {
	EventName() string
	OccurredAt() time.Time
	EventID() uuid.UUID
	SchemaVer() int32
}

// Sentinel error (удобно для errors.Is)
var ErrInvalidEvent = errors.New("invalid event")

// Детализация причины (удобно для логов/диагностики)
var (
	ErrInvalidEventName     = errors.New("invalid event name")
	ErrInvalidEventProducer = errors.New("invalid event producer")
	ErrInvalidEventTime     = errors.New("invalid event time")
	ErrInvalidEventID       = errors.New("invalid event id")
	ErrInvalidEventSchema   = errors.New("invalid event schema version")
)

// BaseEvent — базовая мета-информация любого события.
// Не содержит бизнес-данных и не привязан к Kafka/Transport.
type BaseEvent struct {
	// Core
	Name string
	At   time.Time

	// Idempotency / tracing
	ID            uuid.UUID
	TraceID       string
	CorrelationID string
	CausationID   uuid.UUID

	// Compatibility
	SchemaVersion int32

	// Producer metadata
	Producer string

	// Extensible, non-PII
	Meta map[string]string
}

var _ Event = BaseEvent{} // compile-time contract

// NewBaseEvent — безопасный конструктор (UTC + UUID + schema v1).
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
		At:            time.Now().UTC(),
		ID:            uuid.New(),
		SchemaVersion: 1,
		Producer:      producer,
	}, nil
}

// MustBaseEvent — удобный хелпер для мест, где name/producer константы.
// Используй в конструкторах событий, чтобы не тащить error вверх.
func MustBaseEvent(name, producer string) BaseEvent {
	e, err := NewBaseEvent(name, producer)
	if err != nil {
		panic(err)
	}
	return e
}

// WithTrace — добавить trace/correlation (обычно выставляется в UC из контекста запроса).
func (e BaseEvent) WithTrace(traceID, correlationID string) BaseEvent {
	e.TraceID = strings.TrimSpace(traceID)
	e.CorrelationID = strings.TrimSpace(correlationID)
	return e
}

// WithCausation — указать "что стало причиной" (например, EventID родительского события или CommandID).
func (e BaseEvent) WithCausation(id uuid.UUID) BaseEvent {
	e.CausationID = id
	return e
}

// WithMeta — copy-on-write для карты, чтобы не было скрытого шаринга.
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

	if cur, ok := e.Meta[k]; ok && cur == v {
		return e
	}

	m := make(map[string]string, len(e.Meta)+1)
	maps.Copy(m, e.Meta)
	m[k] = v
	e.Meta = m
	return e
}

func (e BaseEvent) WithSchema(ver int32) BaseEvent {
	if ver > e.SchemaVersion {
		e.SchemaVersion = ver
	}
	return e
}

// Validate — строгая проверка инвариантов события.
// Возвращает ErrInvalidEvent (sentinel) с детализацией причины.
func (e BaseEvent) Validate() error {
	if strings.TrimSpace(e.Name) == "" {
		return fmt.Errorf("%w: %w", ErrInvalidEvent, ErrInvalidEventName)
	}
	if strings.TrimSpace(e.Producer) == "" {
		return fmt.Errorf("%w: %w", ErrInvalidEvent, ErrInvalidEventProducer)
	}
	// At должен быть заполнен и строго в UTC (без локальной локации)
	if e.At.IsZero() || !e.At.Equal(e.At.UTC()) {
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

// Interface implementation

func (e BaseEvent) EventName() string     { return e.Name }
func (e BaseEvent) OccurredAt() time.Time { return e.At } // ожидаем UTC
func (e BaseEvent) EventID() uuid.UUID    { return e.ID }
func (e BaseEvent) SchemaVer() int32      { return e.SchemaVersion }

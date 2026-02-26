package domain

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sync"
)

type EventBuffer struct {
	mu     sync.RWMutex
	events []Event
}

func (b *EventBuffer) Record(e Event) {
	if isNilEvent(e) {
		return
	}
	b.mu.Lock()
	b.events = append(b.events, e)
	b.mu.Unlock()
}

func (b *EventBuffer) RecordStrict(e Event) error {
	if isNilEvent(e) {
		return fmt.Errorf("%w: %w", ErrInvalidEvent, ErrInvalidEventNil)
	}

	if vl, ok := e.(interface{ ValidateWithLimits(EventLimits) error }); ok {
		if err := vl.ValidateWithLimits(DefaultEventLimits); err != nil {
			if errors.Is(err, ErrInvalidEvent) {
				return err
			}
			return fmt.Errorf("%w: %w", ErrInvalidEvent, err)
		}
		b.Record(e)
		return nil
	}

	v, ok := e.(interface{ Validate() error })
	if !ok {
		return fmt.Errorf("%w: event does not expose Validate()", ErrInvalidEvent)
	}

	if err := v.Validate(); err != nil {
		if errors.Is(err, ErrInvalidEvent) {
			return err
		}
		return fmt.Errorf("%w: %w", ErrInvalidEvent, err)
	}

	b.Record(e)
	return nil
}

// Peek returns a snapshot without clearing the buffer.
func (b *EventBuffer) Peek() []Event {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return slices.Clone(b.events)
}

// Pull returns buffered events and clears the buffer.
func (b *EventBuffer) Pull() []Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.events) == 0 {
		return nil
	}
	out := b.events
	b.events = nil
	return out
}

func (b *EventBuffer) Clear() {
	b.mu.Lock()
	b.events = nil
	b.mu.Unlock()
}

func (b *EventBuffer) Len() int {
	b.mu.RLock()
	n := len(b.events)
	b.mu.RUnlock()
	return n
}

func isNilEvent(e Event) bool {
	if e == nil {
		return true
	}
	v := reflect.ValueOf(e)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

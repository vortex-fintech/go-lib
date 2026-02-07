package domain

import "slices"

type EventBuffer struct {
	events []Event
}

func (b *EventBuffer) Record(e Event) {
	if e == nil {
		return
	}
	b.events = append(b.events, e)
}

// Peek — посмотреть без очистки (копия, чтобы снаружи не мутировали slice).
func (b *EventBuffer) Peek() []Event {
	return slices.Clone(b.events)
}

// Pull — забрать и очистить.
func (b *EventBuffer) Pull() []Event {
	if len(b.events) == 0 {
		return nil
	}
	out := b.events
	b.events = nil
	return out
}

func (b *EventBuffer) Clear()   { b.events = nil }
func (b *EventBuffer) Len() int { return len(b.events) }

package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/vortex-fintech/go-lib/foundation/domain"
)

func FuzzBaseEvent_WithMeta_CopyIsolation(f *testing.F) {
	f.Add("trace_id", "abc")
	f.Add("region", "eu")

	f.Fuzz(func(t *testing.T, key, value string) {
		key = strings.TrimSpace(truncateRunes(key, 32))
		value = strings.TrimSpace(truncateRunes(value, 64))
		if key == "" {
			return
		}

		e1 := domain.BaseEvent{Meta: map[string]string{"a": "1"}}
		e2 := e1.WithMeta(key, value)

		e1.Meta["a"] = "mutated"
		if got := e2.Meta["a"]; got != "1" {
			t.Fatalf("copy isolation broken, got %q", got)
		}
	})
}

func FuzzBaseEvent_ValidateWithLimits_NameRules(f *testing.F) {
	f.Add("ok")
	f.Add("   ")
	f.Add("this-name-is-way-too-long-for-limit")

	f.Fuzz(func(t *testing.T, name string) {
		e := domain.BaseEvent{
			Name:          name,
			At:            time.Now().UTC(),
			ID:            uuid.New(),
			SchemaVersion: 1,
			Producer:      "svc",
		}

		limits := domain.EventLimits{
			MaxNameRunes:      8,
			MaxProducerRunes:  16,
			MaxMetaEntries:    2,
			MaxMetaKeyRunes:   8,
			MaxMetaValueRunes: 16,
		}

		err := e.ValidateWithLimits(limits)
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			if !errors.Is(err, domain.ErrInvalidEventName) {
				t.Fatalf("expected invalid name, got %v", err)
			}
			return
		}

		if utf8.RuneCountInString(trimmed) > limits.MaxNameRunes {
			if !errors.Is(err, domain.ErrInvalidEventNameTooLong) {
				t.Fatalf("expected name too long, got %v", err)
			}
			return
		}

		if err != nil {
			t.Fatalf("expected valid event name, got %v", err)
		}
	})
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max])
}

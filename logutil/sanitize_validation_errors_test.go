package logutil

import (
	"reflect"
	"testing"
)

func TestSanitizeValidationErrors_Nil(t *testing.T) {
	got := SanitizeValidationErrors(nil, "", "")
	if got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}

func TestSanitizeValidationErrors_Production_DefaultRedaction(t *testing.T) {
	in := map[string]string{
		"Password": "too short",
		"Email":    "invalid",
		"token":    "abc",
	}
	got := SanitizeValidationErrors(in, "production", "")

	want := map[string]string{
		"Password": "[REDACTED]",
		"Email":    "invalid",
		"token":    "[REDACTED]",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestSanitizeValidationErrors_Production_CustomReplacement(t *testing.T) {
	in := map[string]string{
		"pass": "bad",
	}
	got := SanitizeValidationErrors(in, "production", "***MASK***")

	want := map[string]string{
		"pass": "***MASK***",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestSanitizeValidationErrors_Production_ExtraSensitiveKeys(t *testing.T) {
	in := map[string]string{
		"PIN":   "1234",
		"Email": "invalid",
	}
	got := SanitizeValidationErrors(in, "production", "", "pin")

	want := map[string]string{
		"PIN":   "[REDACTED]",
		"Email": "invalid",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestSanitizeValidationErrors_Development_NoRedaction(t *testing.T) {
	in := map[string]string{
		"Password": "too short",
		"Email":    "invalid",
	}
	got := SanitizeValidationErrors(in, "development", "")

	// В dev ничего не скрываем
	if !reflect.DeepEqual(got, in) {
		t.Fatalf("dev: got %#v, want %#v", got, in)
	}
}

func TestSanitizeValidationErrors_Debug_NoRedaction(t *testing.T) {
	in := map[string]string{
		"token": "abc",
	}
	got := SanitizeValidationErrors(in, "debug", "")

	if !reflect.DeepEqual(got, in) {
		t.Fatalf("debug: got %#v, want %#v", got, in)
	}
}

func TestSanitizeValidationErrors_UnknownEnv_TreatedAsProduction(t *testing.T) {
	in := map[string]string{
		"secret": "x",
		"user":   "ok",
	}
	got := SanitizeValidationErrors(in, "staging", "")

	want := map[string]string{
		"secret": "[REDACTED]",
		"user":   "ok",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unknown env: got %#v, want %#v", got, want)
	}
}

func TestSanitizeValidationErrors_CaseInsensitiveKeys(t *testing.T) {
	in := map[string]string{
		"NewPassword": "weak",
		"SeCrEt":      "x",
		"Email":       "invalid",
	}
	got := SanitizeValidationErrors(in, "production", "")

	want := map[string]string{
		"NewPassword": "[REDACTED]",
		"SeCrEt":      "[REDACTED]",
		"Email":       "invalid",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("case-insensitive: got %#v, want %#v", got, want)
	}
}

func TestSanitizeValidationErrors_InputMapNotMutated(t *testing.T) {
	in := map[string]string{
		"Password": "too short",
		"Email":    "invalid",
	}
	clone := map[string]string{}
	for k, v := range in {
		clone[k] = v
	}

	_ = SanitizeValidationErrors(in, "production", "")

	if !reflect.DeepEqual(in, clone) {
		t.Fatalf("input map mutated: got %#v, want %#v", in, clone)
	}
}

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

	// In development we do not redact values.
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

func TestSanitizeValidationErrors_OutputMapNotAliased(t *testing.T) {
	in := map[string]string{
		"Password": "too short",
		"Email":    "invalid",
	}

	out := SanitizeValidationErrors(in, "development", "")
	out["Email"] = "changed"

	if in["Email"] != "invalid" {
		t.Fatalf("input map must not be affected, got %#v", in)
	}
}

func TestSanitizeValidationErrors_Strict_IgnoresEnvironment(t *testing.T) {
	in := map[string]string{
		"Password": "too short",
		"Email":    "invalid",
	}

	got := SanitizeValidationErrorsStrict(in, "")
	want := map[string]string{
		"Password": "[REDACTED]",
		"Email":    "invalid",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("strict: got %#v, want %#v", got, want)
	}
}

func TestSanitizeValidationErrors_DefaultFintechTokens(t *testing.T) {
	in := map[string]string{
		"user.pin":           "1234",
		"card.cvv":           "123",
		"bank.iban":          "DE89370400440532013000",
		"routing_number":     "021000021",
		"beneficiarySwift":   "DEUTDEFF",
		"profile.account_id": "42",
		"email":              "invalid",
	}

	got := SanitizeValidationErrors(in, "production", "")
	if got["email"] != "invalid" {
		t.Fatalf("email must not be redacted, got %#v", got)
	}

	redactedKeys := []string{
		"user.pin",
		"card.cvv",
		"bank.iban",
		"routing_number",
		"beneficiarySwift",
		"profile.account_id",
	}
	for _, k := range redactedKeys {
		if got[k] != "[REDACTED]" {
			t.Fatalf("expected %s to be redacted, got %#v", k, got)
		}
	}
}

func TestSanitizeValidationErrors_CustomSensitiveTokenMatchesPath(t *testing.T) {
	in := map[string]string{
		"auth.session.id": "s-123",
		"email":           "invalid",
	}

	got := SanitizeValidationErrors(in, "production", "", "session")
	if got["auth.session.id"] != "[REDACTED]" {
		t.Fatalf("expected custom token to redact nested path, got %#v", got)
	}
	if got["email"] != "invalid" {
		t.Fatalf("unexpected redaction for non-sensitive key, got %#v", got)
	}
}

func TestSanitizeValidationErrors_AvoidSubstringFalsePositive(t *testing.T) {
	in := map[string]string{
		"compassion": "nope",
		"passenger":  "name",
	}

	got := SanitizeValidationErrors(in, "production", "")
	if !reflect.DeepEqual(got, in) {
		t.Fatalf("unexpected false positive redaction, got %#v want %#v", got, in)
	}
}

package contactutil

import "strings"

// NormalizeEmail приводит e-mail к нижнему регистру и обрезает пробелы.
// Не валидирует формат, только нормализует.
func NormalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// NormalizeE164 нормализует телефон в формате E.164 — только trim.
// Предполагается, что валидация/формат делаются выше по стеку.
func NormalizeE164(s string) string {
	return strings.TrimSpace(s)
}

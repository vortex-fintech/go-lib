package geo

import "strings"

// NormalizeISO2 приводит ISO 3166-1 alpha-2 код к верхнему регистру
// и обрезает пробелы. Возвращает нормализованный код и ok=false,
// если код некорректный (длина != 2).
func NormalizeISO2(code string) (string, bool) {
	c := strings.ToUpper(strings.TrimSpace(code))
	if len(c) != 2 {
		return "", false
	}
	if c[0] < 'A' || c[0] > 'Z' || c[1] < 'A' || c[1] > 'Z' {
		return "", false
	}
	return c, true
}

// IsValidISO2 проверяет, что код похож на корректный ISO2 (длина == 2
// после trim + upper). Для уже нормализованных значений.
func IsValidISO2(code string) bool {
	_, ok := NormalizeISO2(code)
	return ok
}

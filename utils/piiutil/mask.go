package piiutil

import (
	"strings"
	"unicode"
)

// MaskEmail маскирует локальную часть e-mail, но всегда оставляет что-то видимое.
// Примеры:
//
//	"user@example.com"    -> "u***@example.com"
//	"ab@example.com"      -> "a*@example.com"
//	"u@example.com"       -> "u@example.com"  (одна буква, нечего маскировать)
//	"weird"               -> "w***d"
//	"x"                   -> "x"
func MaskEmail(email string) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return ""
	}

	at := strings.IndexByte(email, '@')
	if at <= 0 {
		// Некорректный email — ведём себя как с обычной строкой:
		// оставляем первую и последнюю букву, середину маскируем.
		runes := []rune(email)
		n := len(runes)
		if n == 1 {
			return string(runes) // "x"
		}
		if n == 2 {
			return string(runes[0]) + "*" // "ab" -> "a*"
		}
		// больше 2 символов
		var b strings.Builder
		b.Grow(n)
		b.WriteRune(runes[0])
		for i := 1; i < n-1; i++ {
			b.WriteRune('*')
		}
		b.WriteRune(runes[n-1])
		return b.String()
	}

	local := email[:at]
	domain := email[at:] // включая '@'
	if len(local) <= 1 {
		// "u@example.com" — нечего маскировать в локальной части
		return local + domain
	}

	// Маскируем всю локальную часть, кроме первого символа.
	// "user" -> "u***"
	runes := []rune(local)
	var b strings.Builder
	b.Grow(len(runes) + len(domain))
	b.WriteRune(runes[0])
	for i := 1; i < len(runes); i++ {
		b.WriteRune('*')
	}
	b.WriteString(domain)
	return b.String()
}

// MaskPhone маскирует телефон, оставляя только последние 1–4 цифры,
// сохраняя формат и спецсимволы (например '+', '-', пробелы).
// Примеры:
//
//	"+1234567890"       -> "+******7890"
//	"+1234"             -> "+***4"
//	"123"               -> "**3"
//	"12"                -> "*2"
//	"1"                 -> "1"
//	"AB-CD" (нет цифр)  -> "*B-CD" (маскируем буквы, кроме последней значимой)
func MaskPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return ""
	}

	runes := []rune(phone)

	// считаем цифры
	totalDigits := 0
	for _, r := range runes {
		if unicode.IsDigit(r) {
			totalDigits++
		}
	}

	if totalDigits == 0 {
		// Вообще нет цифр — рассматриваем как ID: маскируем все буквы/цифры, кроме последней
		return maskLettersAndDigitsKeepLast(runes, 1)
	}

	// Сколько цифр оставить:
	//  - если цифр > 4 → оставляем 4 последние
	//  - если цифр <= 4 → оставляем 1 последнюю
	keepDigits := 4
	if totalDigits < 4 {
		keepDigits = 1
	}

	digitsSeen := 0
	for i := len(runes) - 1; i >= 0; i-- {
		if unicode.IsDigit(runes[i]) {
			digitsSeen++
			if digitsSeen > keepDigits {
				runes[i] = '*'
			}
		}
	}
	return string(runes)
}

// MaskIDLast4 маскирует идентификатор (TaxID, SSN, NRIC и т.п.), оставляя
// только последние 1–4 цифры, сохраняя разделители. Если цифр нет —
// маскирует буквы/цифры, оставляя последних N значимых символов.
//
// Примеры:
//
//	"123-45-6789"    -> "***-**-6789"
//	"S1234567D"      -> "S***4567D"
//	"AB-1234-CD"     -> "AB-***4-CD"   (цифр мало, оставляем 1 последнюю)
//	"12-AB"          -> "*2-AB"
//	"ABCD"           -> "***D"
//	"X"              -> "X"
func MaskIDLast4(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	runes := []rune(s)

	// считаем цифры
	totalDigits := 0
	for _, r := range runes {
		if unicode.IsDigit(r) {
			totalDigits++
		}
	}

	if totalDigits == 0 {
		// Нет цифр — маскируем все буквы/цифры, кроме 1 последней "значимой"
		return maskLettersAndDigitsKeepLast(runes, 1)
	}

	// Если цифр мало, показываем только 1 последнюю,
	// иначе показываем 4 последних.
	keepDigits := 4
	if totalDigits < 4 {
		keepDigits = 1
	}

	digitsSeen := 0
	for i := len(runes) - 1; i >= 0; i-- {
		if unicode.IsDigit(runes[i]) {
			digitsSeen++
			if digitsSeen > keepDigits {
				runes[i] = '*'
			}
		}
	}
	return string(runes)
}

// Вспомогательная функция: маскирует все буквы/цифры, кроме последних keep значимых.
func maskLettersAndDigitsKeepLast(runes []rune, keep int) string {
	n := len(runes)
	if n == 0 {
		return ""
	}
	if keep < 1 {
		keep = 1
	}

	// Считаем "значимые" (буквы/цифры)
	total := 0
	for _, r := range runes {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			total++
		}
	}
	if total == 0 {
		// только разделители — ничего не маскируем
		return string(runes)
	}
	if keep > total {
		keep = total
	}

	// идём справа налево и оставляем последние keep значимых, остальные маскируем
	seen := 0
	for i := len(runes) - 1; i >= 0; i-- {
		r := runes[i]
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			seen++
			if seen > keep {
				runes[i] = '*'
			}
		}
	}
	return string(runes)
}

package authz

// MapResolver — простой резолвер политик по полному имени метода.
// Если метод не найден — возвращает пустую политику.
func MapResolver(m map[string]Policy) PolicyResolver {
	return func(fullMethod string) Policy {
		if p, ok := m[fullMethod]; ok {
			return p
		}
		return Policy{}
	}
}

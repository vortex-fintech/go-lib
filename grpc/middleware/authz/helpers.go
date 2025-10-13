package authz

// MapResolver — резолвер политик по полному имени метода.
func MapResolver(m map[string]Policy) PolicyResolver {
	return func(fullMethod string) Policy {
		if p, ok := m[fullMethod]; ok {
			return p
		}
		return Policy{}
	}
}

// MapSkipAuth — пропустить аутентификацию для методов из карты.
func MapSkipAuth(allow map[string]struct{}) SkipAuthFunc {
	return func(fullMethod string) bool {
		_, ok := allow[fullMethod]
		return ok
	}
}

// SliceSkipAuth — удобный helper на базе списка методов.
func SliceSkipAuth(methods ...string) SkipAuthFunc {
	m := make(map[string]struct{}, len(methods))
	for _, s := range methods {
		m[s] = struct{}{}
	}
	return MapSkipAuth(m)
}

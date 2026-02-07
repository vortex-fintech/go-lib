package scope

// Simple helpers to evaluate scopes declared in JWT claims.

func Index(scopes []string) map[string]struct{} {
	m := make(map[string]struct{}, len(scopes))
	for _, s := range scopes {
		m[s] = struct{}{}
	}
	return m
}

func HasAll(scopes []string, need ...string) bool {
	if len(need) == 0 {
		return true
	}
	m := Index(scopes)
	for _, n := range need {
		if _, ok := m[n]; !ok {
			return false
		}
	}
	return true
}

func HasAny(scopes []string, any ...string) bool {
	if len(any) == 0 {
		return true
	}
	m := Index(scopes)
	for _, n := range any {
		if _, ok := m[n]; ok {
			return true
		}
	}
	return false
}

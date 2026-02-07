package validator

var tagMap = map[string]string{
	"required":  "required",
	"omitempty": "optional",
	"email":     "invalid_email",
	"e164":      "invalid_phone",
	"uuid4":     "invalid_uuid",
	"uuid":      "invalid_uuid",
	"ip":        "invalid_ip",
	"ipv4":      "invalid_ipv4",
	"ipv6":      "invalid_ipv6",
	"url":       "invalid_url",
	"http_url":  "invalid_http_url",
	"eqfield":   "field_mismatch",
	"nefield":   "field_should_differ",
	"eq":        "not_equal_to_required_value",
	"ne":        "should_not_equal_value",
	"max":       "too_long",
	"min":       "too_short",
	"gt":        "too_small",
	"lt":        "too_large",
	"gte":       "too_small_or_equal",
	"lte":       "too_large_or_equal",
	"len":       "invalid_length",
	"oneof":     "invalid_choice",
	"contains":  "missing_required_substring",
	"excludes":  "should_not_contain",
	"alpha":     "only_letters_allowed",
	"alphanum":  "only_letters_and_digits_allowed",
	"numeric":   "only_numbers_allowed",
	"boolean":   "invalid_boolean",
}

func mapTagToCode(tag string) string {
	if code, ok := tagMap[tag]; ok {
		return code
	}
	return "invalid"
}

package validator

import "github.com/go-playground/validator/v10"

var v *validator.Validate

func init() {
	v = validator.New()
}

func Instance() *validator.Validate {
	return v
}

func Validate(i any) map[string]string {
	if err := v.Struct(i); err != nil {
		if errs, ok := err.(validator.ValidationErrors); ok {
			out := make(map[string]string)
			for _, e := range errs {
				out[e.Field()] = mapTagToCode(e.Tag())
			}
			return out
		}
		return map[string]string{"_error": "validation_failed"}
	}
	return nil
}

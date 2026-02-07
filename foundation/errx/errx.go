package errx

import (
	"fmt"
	"strings"
)

func State(base error, msg string) error {
	return wrap("state", base, msg)
}

func Transition(base error, msg string) error {
	return wrap("transition", base, msg)
}

func wrap(kind string, base error, msg string) error {
	msg = strings.TrimSpace(msg)
	switch {
	case base == nil && msg == "":
		return fmt.Errorf("%s error", kind)
	case base == nil:
		return fmt.Errorf("%s: %s", kind, msg)
	case msg == "":
		return fmt.Errorf("%s: %w", kind, base)
	default:
		return fmt.Errorf("%s: %w: %s", kind, base, msg)
	}
}

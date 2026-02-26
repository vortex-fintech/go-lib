package postgres

import "context"

type ctxKeyRunner struct{}

// ContextWithRunner stores a Runner in the context.
func ContextWithRunner(ctx context.Context, r Runner) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxKeyRunner{}, r)
}

// RunnerFromContext extracts the Runner from the context.
// Falls back to pool runner if no transaction is active.
func RunnerFromContext(ctx context.Context, fallback *Client) Runner {
	if ctx != nil {
		if r, ok := ctx.Value(ctxKeyRunner{}).(Runner); ok {
			return r
		}
	}
	if fallback == nil {
		return nil
	}
	return fallback.RunnerFromPool()
}

// MustRunnerFromContext extracts the Runner or panics.
func MustRunnerFromContext(ctx context.Context) Runner {
	if ctx == nil {
		panic("postgres: no Runner in context (outside transaction?)")
	}
	r, ok := ctx.Value(ctxKeyRunner{}).(Runner)
	if !ok {
		panic("postgres: no Runner in context (outside transaction?)")
	}
	return r
}

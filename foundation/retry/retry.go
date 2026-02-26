package retry

import (
	"context"
	"errors"
	"time"

	"github.com/cenkalti/backoff/v5"
)

const (
	defaultInitInitialInterval = 500 * time.Millisecond
	defaultInitMultiplier      = 2.0
	defaultInitMaxInterval     = 5 * time.Second
	defaultInitRandomization   = 0.5
	defaultInitMaxElapsed      = 20 * time.Second

	defaultFastMaxAttempts = 3
	defaultFastDelay       = 200 * time.Millisecond
)

// PermanentError wraps a non-retryable error.
type PermanentError struct {
	err error
}

func (e PermanentError) Error() string {
	if e.err == nil {
		return "permanent error"
	}
	return e.err.Error()
}

func (e PermanentError) Unwrap() error { return e.err }

// Permanent marks an error as non-retryable.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	if IsPermanent(err) {
		return err
	}
	return PermanentError{err: err}
}

// IsPermanent reports whether err is marked as non-retryable.
func IsPermanent(err error) bool {
	var pe PermanentError
	if errors.As(err, &pe) {
		return true
	}

	var bpe *backoff.PermanentError
	return errors.As(err, &bpe)
}

// RetryInit retries fn with exponential backoff for startup/init flows.
// It stops on context cancellation, permanent errors, or max elapsed time.
func RetryInit(ctx context.Context, fn func() error) error {
	exp := backoff.NewExponentialBackOff()
	exp.InitialInterval = defaultInitInitialInterval
	exp.Multiplier = defaultInitMultiplier
	exp.MaxInterval = defaultInitMaxInterval
	exp.RandomizationFactor = defaultInitRandomization
	exp.Reset()

	type unit struct{}
	op := func() (unit, error) {
		if err := ctx.Err(); err != nil {
			return unit{}, err
		}

		err := fn()
		if IsPermanent(err) {
			var bpe *backoff.PermanentError
			if errors.As(err, &bpe) {
				return unit{}, err
			}
			return unit{}, backoff.Permanent(err)
		}
		return unit{}, err
	}

	_, err := backoff.Retry(
		ctx,
		op,
		backoff.WithBackOff(exp),
		backoff.WithMaxElapsedTime(defaultInitMaxElapsed),
	)
	return err
}

// RetryFast retries fn a small fixed number of times for short transient failures.
// It stops on context cancellation or permanent errors.
func RetryFast(ctx context.Context, fn func() error) error {
	var err error
	for i := 0; i < defaultFastMaxAttempts; i++ {
		if err = ctx.Err(); err != nil {
			return err
		}

		err = fn()
		if err == nil {
			return nil
		}
		if IsPermanent(err) {
			return err
		}
		if i == defaultFastMaxAttempts-1 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(defaultFastDelay):
		}
	}
	return err
}

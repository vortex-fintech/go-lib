package retry

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v5"
)

func RetryInit(ctx context.Context, fn func() error) error {
	exp := backoff.NewExponentialBackOff()
	exp.InitialInterval = 500 * time.Millisecond
	exp.Multiplier = 2.0
	exp.MaxInterval = 5 * time.Second
	exp.RandomizationFactor = 0.5
	exp.Reset()

	type unit struct{}
	op := func() (unit, error) {
		return unit{}, fn()
	}

	_, err := backoff.Retry(
		ctx,
		op,
		backoff.WithBackOff(exp),
		backoff.WithMaxElapsedTime(20*time.Second),
	)
	return err
}

func RetryFast(ctx context.Context, fn func() error) error {
	const (
		maxAttempts = 3
		delay       = 200 * time.Millisecond
	)

	var err error
	for i := 0; i < maxAttempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return err
}

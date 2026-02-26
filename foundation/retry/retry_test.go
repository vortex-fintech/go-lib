//go:build unit
// +build unit

package retry_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/stretchr/testify/assert"
	"github.com/vortex-fintech/go-lib/foundation/retry"
)

func TestPermanentNilReturnsNil(t *testing.T) {
	assert.NoError(t, retry.Permanent(nil))
}

func TestPermanentIdempotent(t *testing.T) {
	base := errors.New("invalid")
	first := retry.Permanent(base)
	second := retry.Permanent(first)

	assert.Equal(t, first, second)
	assert.True(t, retry.IsPermanent(second))
	assert.ErrorIs(t, second, base)
}

func TestPermanentErrorZeroValue(t *testing.T) {
	var pe retry.PermanentError
	assert.Equal(t, "permanent error", pe.Error())
	assert.Nil(t, pe.Unwrap())
}

func TestRetryInit_Success(t *testing.T) {
	ctx := context.Background()
	calls := 0
	err := retry.RetryInit(ctx, func() error {
		calls++
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestRetryInit_Fail(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	calls := 0
	err := retry.RetryInit(ctx, func() error {
		calls++
		return errors.New("fail")
	})
	assert.Error(t, err)
	assert.Greater(t, calls, 1)
}

func TestRetryInit_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	calls := 0
	err := retry.RetryInit(ctx, func() error {
		calls++
		return errors.New("fail")
	})
	assert.Error(t, err)
	assert.True(t, ctx.Err() != nil)
}

func TestRetryFast_Success(t *testing.T) {
	ctx := context.Background()
	calls := 0
	err := retry.RetryFast(ctx, func() error {
		calls++
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestRetryFast_Fail(t *testing.T) {
	ctx := context.Background()
	calls := 0
	err := retry.RetryFast(ctx, func() error {
		calls++
		return errors.New("fail")
	})
	assert.Error(t, err)
	assert.Equal(t, 3, calls)
}

func TestRetryFast_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	calls := 0
	err := retry.RetryFast(ctx, func() error {
		calls++
		time.Sleep(60 * time.Millisecond)
		return errors.New("fail")
	})
	assert.Error(t, err)
	assert.True(t, ctx.Err() != nil)
	assert.GreaterOrEqual(t, calls, 1)
}

func TestRetryFast_ContextAlreadyCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	calls := 0
	err := retry.RetryFast(ctx, func() error {
		calls++
		return errors.New("fail")
	})

	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 0, calls)
}

func TestRetryFast_NoExtraDelayAfterLastAttempt(t *testing.T) {
	ctx := context.Background()
	start := time.Now()

	err := retry.RetryFast(ctx, func() error {
		return errors.New("fail")
	})
	elapsed := time.Since(start)

	assert.Error(t, err)
	assert.Less(t, elapsed, 550*time.Millisecond)
}

func TestRetryFast_PermanentStopsImmediately(t *testing.T) {
	ctx := context.Background()
	base := errors.New("bad request")
	calls := 0

	err := retry.RetryFast(ctx, func() error {
		calls++
		return retry.Permanent(base)
	})

	assert.Error(t, err)
	assert.True(t, retry.IsPermanent(err))
	assert.ErrorIs(t, err, base)
	assert.Equal(t, 1, calls)
}

func TestRetryFast_BackoffPermanentStopsImmediately(t *testing.T) {
	ctx := context.Background()
	base := errors.New("no point retrying")
	calls := 0

	err := retry.RetryFast(ctx, func() error {
		calls++
		return backoff.Permanent(base)
	})

	assert.Error(t, err)
	assert.True(t, retry.IsPermanent(err))
	assert.ErrorIs(t, err, base)
	assert.Equal(t, 1, calls)
}

func TestRetryInit_PermanentStopsImmediately(t *testing.T) {
	ctx := context.Background()
	base := errors.New("invalid credentials")
	calls := 0

	err := retry.RetryInit(ctx, func() error {
		calls++
		return retry.Permanent(base)
	})

	assert.Error(t, err)
	assert.True(t, retry.IsPermanent(err))
	assert.ErrorIs(t, err, base)
	assert.Equal(t, 1, calls)
}

func TestRetryInit_BackoffPermanentStopsImmediately(t *testing.T) {
	ctx := context.Background()
	base := errors.New("unrecoverable")
	calls := 0

	err := retry.RetryInit(ctx, func() error {
		calls++
		return backoff.Permanent(base)
	})

	assert.Error(t, err)
	assert.True(t, retry.IsPermanent(err))
	assert.ErrorIs(t, err, base)
	assert.Equal(t, 1, calls)
}

func TestIsPermanent_Wrapped(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", retry.Permanent(errors.New("invalid")))
	assert.True(t, retry.IsPermanent(err))
}

func TestIsPermanent_BackoffWrapped(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", backoff.Permanent(errors.New("invalid")))
	assert.True(t, retry.IsPermanent(err))
}

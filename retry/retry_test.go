//go:build unit
// +build unit

package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vortex-fintech/go-lib/retry"
)

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
	ctx := context.Background()
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

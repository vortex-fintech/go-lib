//go:build unit
// +build unit

package logger

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestBuildConfigByEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		env                 string
		wantLevel           zapcore.Level
		wantDisableStack    bool
		wantCaller          bool
		wantCallerKey       string
		wantColorLevelCodec bool
	}{
		{
			name:                "development",
			env:                 "development",
			wantLevel:           zap.DebugLevel,
			wantDisableStack:    true,
			wantCaller:          false,
			wantCallerKey:       zapcore.OmitKey,
			wantColorLevelCodec: true,
		},
		{
			name:                "debug",
			env:                 " DEBUG ",
			wantLevel:           zap.DebugLevel,
			wantDisableStack:    false,
			wantCaller:          true,
			wantCallerKey:       "caller",
			wantColorLevelCodec: true,
		},
		{
			name:                "production",
			env:                 "production",
			wantLevel:           zap.InfoLevel,
			wantDisableStack:    true,
			wantCaller:          false,
			wantCallerKey:       zapcore.OmitKey,
			wantColorLevelCodec: false,
		},
		{
			name:                "fallback",
			env:                 "unknown",
			wantLevel:           zap.InfoLevel,
			wantDisableStack:    true,
			wantCaller:          false,
			wantCallerKey:       zapcore.OmitKey,
			wantColorLevelCodec: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg, withCaller := buildConfig(tc.env)

			require.Equal(t, tc.wantLevel, cfg.Level.Level())
			require.Equal(t, tc.wantDisableStack, cfg.DisableStacktrace)
			require.Equal(t, tc.wantCaller, withCaller)
			require.Equal(t, tc.wantCallerKey, cfg.EncoderConfig.CallerKey)
			require.Equal(t, "timestamp", cfg.EncoderConfig.TimeKey)
			require.Equal(t, "level", cfg.EncoderConfig.LevelKey)
			require.Equal(t, "msg", cfg.EncoderConfig.MessageKey)
			require.Equal(t, "logger", cfg.EncoderConfig.NameKey)
			require.Equal(t, []string{"stdout"}, cfg.OutputPaths)

			if tc.wantColorLevelCodec {
				require.NotNil(t, cfg.EncoderConfig.EncodeLevel)
			}
		})
	}
}

func TestNewReturnsLogger(t *testing.T) {
	t.Parallel()

	l, err := New("svc", "production")
	require.NoError(t, err)
	require.NotNil(t, l)

	l.Infow("startup", "component", "logger")
	l.SafeSync()
}

func TestIsIgnorableSyncError(t *testing.T) {
	t.Parallel()

	require.False(t, isIgnorableSyncError(nil))
	require.True(t, isIgnorableSyncError(errors.New("sync /dev/stdout: invalid argument")))
	require.True(t, isIgnorableSyncError(errors.New("sync /dev/stdout: inappropriate ioctl for device")))
	require.False(t, isIgnorableSyncError(errors.New("disk write failed")))
}

func TestContextHelpers_NilContext(t *testing.T) {
	t.Parallel()

	ctx := ContextWithTraceID(nil, "trace-1")
	if ctx == nil {
		t.Fatalf("expected non-nil context")
	}
	v, ok := ctx.Value(traceIDKey).(string)
	if !ok || v != "trace-1" {
		t.Fatalf("unexpected trace id in context: %v", ctx.Value(traceIDKey))
	}

	ctx = ContextWithRequestID(nil, "req-1")
	if ctx == nil {
		t.Fatalf("expected non-nil context")
	}
	v, ok = ctx.Value(requestIDKey).(string)
	if !ok || v != "req-1" {
		t.Fatalf("unexpected request id in context: %v", ctx.Value(requestIDKey))
	}

	base := context.Background()
	ctx = ContextWithTraceID(base, "trace-2")
	ctx = ContextWithRequestID(ctx, "req-2")
	if ctx.Value(traceIDKey) != "trace-2" || ctx.Value(requestIDKey) != "req-2" {
		t.Fatalf("expected both ids in context")
	}
}

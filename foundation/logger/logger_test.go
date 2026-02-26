//go:build unit
// +build unit

package logger_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vortex-fintech/go-lib/foundation/logger"
)

func TestInitAndBasicMethods(t *testing.T) {
	log := logger.Init("test-service", "development")
	assert.NotNil(t, log)

	log.Info("info")
	log.Warn("warn")
	log.Error("error")
	log.Debug("debug")

	log.Infof("infof: %s", "ok")
	log.Warnf("warnf: %s", "ok")
	log.Errorf("errorf: %s", "ok")
	log.Debugf("debugf: %s", "ok")

	log.Infow("infow", "key", "value")
	log.Warnw("warnw", "key", "value")
	log.Errorw("errorw", "key", "value")
	log.Debugw("debugw", "key", "value")

	l2 := log.With("user", "test")
	assert.NotNil(t, l2)
	l2.Info("with works")

	log.SafeSync()
}

func TestInitEnvs(t *testing.T) {
	for _, env := range []string{"development", "debug", "production", "unknown"} {
		log := logger.Init("svc", env)
		assert.NotNil(t, log)
		log.Info("env:", env)
	}
}

func TestCtxMethods_ExtractTraceAndRequestID(t *testing.T) {
	log, err := logger.New("test-service", "production")
	require.NoError(t, err)

	ctx := context.Background()
	ctx = logger.ContextWithTraceID(ctx, "trace-123")
	ctx = logger.ContextWithRequestID(ctx, "req-456")

	log.InfowCtx(ctx, "message with context", "user_id", "user-789")
	log.WarnwCtx(ctx, "warning with context")
	log.ErrorwCtx(ctx, "error with context")
	log.DebugwCtx(ctx, "debug with context")

	log.SafeSync()
}

func TestCtxMethods_NilContext(t *testing.T) {
	log, err := logger.New("test-service", "production")
	require.NoError(t, err)

	log.InfowCtx(nil, "message without context", "key", "value")
	log.SafeSync()
}

func TestCtxMethods_EmptyContext(t *testing.T) {
	log, err := logger.New("test-service", "production")
	require.NoError(t, err)

	ctx := context.Background()
	log.InfowCtx(ctx, "message with empty context")
	log.SafeSync()
}

func TestCtxMethods_TraceIDOnly(t *testing.T) {
	log, err := logger.New("test-service", "production")
	require.NoError(t, err)

	ctx := logger.ContextWithTraceID(context.Background(), "trace-only")
	log.InfowCtx(ctx, "trace only")
	log.SafeSync()
}

func TestCtxMethods_RequestIDOnly(t *testing.T) {
	log, err := logger.New("test-service", "production")
	require.NoError(t, err)

	ctx := logger.ContextWithRequestID(context.Background(), "req-only")
	log.InfowCtx(ctx, "request only")
	log.SafeSync()
}

func TestLoggerInterface_Compliance(t *testing.T) {
	log, err := logger.New("test-service", "production")
	require.NoError(t, err)

	var _ logger.LoggerInterface = log
	var _ logger.LoggerInterface = log.With("key", "value")
}

func TestWithChaining(t *testing.T) {
	log, err := logger.New("test-service", "production")
	require.NoError(t, err)

	l1 := log.With("service", "payments")
	l2 := l1.With("version", "1.0.0")

	ctx := logger.ContextWithTraceID(context.Background(), "trace-abc")
	l2.InfowCtx(ctx, "chained with context", "extra", "field")

	log.SafeSync()
}

func TestStructuredOutputContainsFields(t *testing.T) {
	log, err := logger.New("test-service", "production")
	require.NoError(t, err)

	ctx := logger.ContextWithTraceID(context.Background(), "trace-xyz")
	ctx = logger.ContextWithRequestID(ctx, "req-xyz")

	log.InfowCtx(ctx, "structured test", "amount", 1000, "currency", "USD")
	log.SafeSync()
}

func TestFatalMethods_Exist(t *testing.T) {
	log, err := logger.New("test-service", "production")
	require.NoError(t, err)

	assert.NotNil(t, log.Fatal)
	assert.NotNil(t, log.Fatalf)
	assert.NotNil(t, log.Fatalw)
	assert.NotNil(t, log.FatalwCtx)
}

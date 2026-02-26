package logger

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ctxKey string

const (
	traceIDKey   ctxKey = "trace_id"
	requestIDKey ctxKey = "request_id"
)

type Logger struct {
	*zap.SugaredLogger
}

func Init(serviceName, env string) *Logger {
	l, err := New(serviceName, env)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	return l
}

func New(serviceName, env string) (*Logger, error) {
	cfg, withCaller := buildConfig(env)

	z, err := cfg.Build(
		zap.WithCaller(withCaller),
		zap.AddCallerSkip(1),
	)
	if err != nil {
		return nil, fmt.Errorf("cannot init zap logger: %w", err)
	}

	return &Logger{SugaredLogger: z.Named(serviceName).Sugar()}, nil
}

func buildConfig(env string) (zap.Config, bool) {
	var cfg zap.Config
	withCaller := false

	switch strings.ToLower(strings.TrimSpace(env)) {
	case "development":
		cfg = zap.NewDevelopmentConfig()
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.DisableStacktrace = true

	case "debug":
		cfg = zap.NewDevelopmentConfig()
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.DisableStacktrace = false
		withCaller = true

	case "production":
		cfg = zap.NewProductionConfig()
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
		cfg.DisableStacktrace = true

	default:
		cfg = zap.NewDevelopmentConfig()
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.DisableStacktrace = true
	}

	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.LevelKey = "level"
	cfg.EncoderConfig.MessageKey = "msg"
	cfg.EncoderConfig.NameKey = "logger"
	if withCaller {
		cfg.EncoderConfig.CallerKey = "caller"
	} else {
		cfg.EncoderConfig.CallerKey = zapcore.OmitKey
	}
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.OutputPaths = []string{"stdout"}

	return cfg, withCaller
}

func (l *Logger) With(args ...any) LoggerInterface {
	return &Logger{SugaredLogger: l.SugaredLogger.With(args...)}
}

func (l *Logger) SafeSync() {
	if l == nil {
		return
	}
	if err := l.Desugar().Sync(); err != nil {
		if !isIgnorableSyncError(err) {
			l.Errorf("log sync error: %v", err)
		}
	}
}

func isIgnorableSyncError(err error) bool {
	if err == nil {
		return false
	}

	s := strings.ToLower(err.Error())
	return strings.Contains(s, "invalid argument") ||
		strings.Contains(s, "inappropriate ioctl for device")
}

func (l *Logger) Info(a ...any)  { l.SugaredLogger.Info(a...) }
func (l *Logger) Warn(a ...any)  { l.SugaredLogger.Warn(a...) }
func (l *Logger) Error(a ...any) { l.SugaredLogger.Error(a...) }
func (l *Logger) Debug(a ...any) { l.SugaredLogger.Debug(a...) }
func (l *Logger) Fatal(a ...any) { l.SugaredLogger.Fatal(a...) }

func (l *Logger) Infof(t string, a ...any)  { l.SugaredLogger.Infof(t, a...) }
func (l *Logger) Warnf(t string, a ...any)  { l.SugaredLogger.Warnf(t, a...) }
func (l *Logger) Errorf(t string, a ...any) { l.SugaredLogger.Errorf(t, a...) }
func (l *Logger) Debugf(t string, a ...any) { l.SugaredLogger.Debugf(t, a...) }
func (l *Logger) Fatalf(t string, a ...any) { l.SugaredLogger.Fatalf(t, a...) }

func (l *Logger) Infow(m string, kv ...any)  { l.SugaredLogger.Infow(m, kv...) }
func (l *Logger) Warnw(m string, kv ...any)  { l.SugaredLogger.Warnw(m, kv...) }
func (l *Logger) Errorw(m string, kv ...any) { l.SugaredLogger.Errorw(m, kv...) }
func (l *Logger) Fatalw(m string, kv ...any) { l.SugaredLogger.Fatalw(m, kv...) }
func (l *Logger) Debugw(m string, kv ...any) { l.SugaredLogger.Debugw(m, kv...) }

func (l *Logger) InfowCtx(ctx context.Context, msg string, kv ...any) {
	l.SugaredLogger.Infow(msg, appendContextFields(ctx, kv...)...)
}

func (l *Logger) WarnwCtx(ctx context.Context, msg string, kv ...any) {
	l.SugaredLogger.Warnw(msg, appendContextFields(ctx, kv...)...)
}

func (l *Logger) ErrorwCtx(ctx context.Context, msg string, kv ...any) {
	l.SugaredLogger.Errorw(msg, appendContextFields(ctx, kv...)...)
}

func (l *Logger) DebugwCtx(ctx context.Context, msg string, kv ...any) {
	l.SugaredLogger.Debugw(msg, appendContextFields(ctx, kv...)...)
}

func (l *Logger) FatalwCtx(ctx context.Context, msg string, kv ...any) {
	l.SugaredLogger.Fatalw(msg, appendContextFields(ctx, kv...)...)
}

func appendContextFields(ctx context.Context, kv ...any) []any {
	if ctx == nil {
		return kv
	}

	if v := ctx.Value(traceIDKey); v != nil {
		if s, ok := v.(string); ok && s != "" {
			kv = append(kv, "trace_id", s)
		}
	}

	if v := ctx.Value(requestIDKey); v != nil {
		if s, ok := v.(string); ok && s != "" {
			kv = append(kv, "request_id", s)
		}
	}

	return kv
}

func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, traceIDKey, traceID)
}

func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, requestIDKey, requestID)
}

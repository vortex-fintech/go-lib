package logger

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.SugaredLogger
}

func Init(serviceName, env string) *Logger {
	cfg := buildConfig(env)

	z, err := cfg.Build(
		zap.WithCaller(false),
		zap.AddCallerSkip(1),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot init zap logger: %v\n", err)
		os.Exit(1)
	}

	return &Logger{SugaredLogger: z.Named(serviceName).Sugar()}
}

func buildConfig(env string) zap.Config {
	var cfg zap.Config

	switch env {
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
		cfg.EncoderConfig.CallerKey = "caller"

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
	cfg.EncoderConfig.CallerKey = zapcore.OmitKey
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.OutputPaths = []string{"stdout"}

	return cfg
}

func (l *Logger) With(args ...any) LoggerInterface {
	return &Logger{SugaredLogger: l.SugaredLogger.With(args...)}
}

func (l *Logger) SafeSync() {
	if l == nil {
		return
	}
	if err := l.Desugar().Sync(); err != nil {
		if !strings.Contains(err.Error(), "invalid argument") {
			l.Errorf("log sync error: %v", err)
		}
	}
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

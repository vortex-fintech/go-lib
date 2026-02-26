package logger

import "context"

type LoggerInterface interface {
	Info(...any)
	Warn(...any)
	Error(...any)
	Debug(...any)
	Fatal(...any)

	Infof(string, ...any)
	Warnf(string, ...any)
	Errorf(string, ...any)
	Debugf(string, ...any)
	Fatalf(string, ...any)

	Infow(string, ...any)
	Warnw(string, ...any)
	Errorw(string, ...any)
	Debugw(string, ...any)
	Fatalw(string, ...any)

	InfowCtx(context.Context, string, ...any)
	WarnwCtx(context.Context, string, ...any)
	ErrorwCtx(context.Context, string, ...any)
	DebugwCtx(context.Context, string, ...any)
	FatalwCtx(context.Context, string, ...any)

	With(...any) LoggerInterface
	SafeSync()
}

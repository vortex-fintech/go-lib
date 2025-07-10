package logger

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

	With(...any) LoggerInterface
	SafeSync()
}

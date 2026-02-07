package circuitbreaker

import "github.com/vortex-fintech/go-lib/foundation/logger"

type GoLibLoggerAdapter struct{ L logger.LoggerInterface }

func (a GoLibLoggerAdapter) Info(msg string)  { a.L.Info(msg) }
func (a GoLibLoggerAdapter) Warn(msg string)  { a.L.Warn(msg) }
func (a GoLibLoggerAdapter) Error(msg string) { a.L.Error(msg) }

func WithGoLibLogger(l logger.LoggerInterface) Option {
	return WithLogger(GoLibLoggerAdapter{L: l})
}

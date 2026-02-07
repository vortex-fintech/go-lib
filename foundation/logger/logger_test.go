//go:build unit
// +build unit

package logger_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

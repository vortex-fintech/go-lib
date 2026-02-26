package mtls

import (
	"errors"
	"log/slog"
	"os"
	"sync"
	"time"
)

var errNilTicker = errors.New("mtls: ticker is nil")

type ReloadEvent struct {
	Err error
}

type ReloadLogger func(ev ReloadEvent)

func DefaultReloadLogger(ev ReloadEvent) {
	if ev.Err != nil {
		slog.Error("mtls: reload failed", "error", ev.Err)
	} else {
		slog.Info("mtls: certificates reloaded")
	}
}

type Reloader struct {
	cfg      Config
	lastCA   time.Time
	lastCrt  time.Time
	lastKey  time.Time
	apply    func(*bundle)
	log      ReloadLogger
	stopCh   chan struct{}
	stopOnce sync.Once
}

func NewReloader(cfg Config, apply func(*bundle)) *Reloader {
	return NewReloaderWithLogger(cfg, apply, DefaultReloadLogger)
}

func NewReloaderWithLogger(cfg Config, apply func(*bundle), log ReloadLogger) *Reloader {
	if apply == nil {
		apply = func(*bundle) {}
	}
	if log == nil {
		log = func(ReloadEvent) {}
	}
	return &Reloader{cfg: cfg, apply: apply, log: log, stopCh: make(chan struct{})}
}

func (r *Reloader) Start(t *time.Ticker) {
	if r == nil {
		return
	}
	if t == nil {
		r.log(ReloadEvent{Err: errNilTicker})
		return
	}
	r.snap()
	go func() {
		for {
			select {
			case <-t.C:
				if r.changed() {
					if nb, err := loadBundle(r.cfg); err == nil {
						r.snap()
						r.apply(nb)
						r.log(ReloadEvent{})
					} else {
						r.log(ReloadEvent{Err: err})
					}
				}
			case <-r.stopCh:
				t.Stop()
				return
			}
		}
	}()
}

func (r *Reloader) Stop() {
	if r == nil {
		return
	}
	r.stopOnce.Do(func() {
		close(r.stopCh)
	})
}

func (r *Reloader) snap() {
	r.lastCA = mtime(r.cfg.CACertPath)
	r.lastCrt = mtime(r.cfg.CertPath)
	r.lastKey = mtime(r.cfg.KeyPath)
}

func (r *Reloader) changed() bool {
	return !mtime(r.cfg.CACertPath).Equal(r.lastCA) ||
		!mtime(r.cfg.CertPath).Equal(r.lastCrt) ||
		!mtime(r.cfg.KeyPath).Equal(r.lastKey)
}

func mtime(path string) time.Time {
	if fi, err := os.Stat(path); err == nil {
		return fi.ModTime()
	}
	return time.Time{}
}

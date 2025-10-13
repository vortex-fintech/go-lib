package mtls

import (
	"log"
	"os"
	"time"
)

// Reloader is a tiny polling-based cert reloader to avoid external deps.
type Reloader struct {
	cfg     Config
	lastCA  time.Time
	lastCrt time.Time
	lastKey time.Time
	apply   func(*bundle) // called when new bundle is ready
	stopCh  chan struct{}
}

func NewReloader(cfg Config, apply func(*bundle)) *Reloader {
	return &Reloader{cfg: cfg, apply: apply, stopCh: make(chan struct{})}
}

// Start begins periodic checks. Provide a ticker; caller owns it.
func (r *Reloader) Start(t *time.Ticker) {
	// init modified times
	r.snap()
	go func() {
		for {
			select {
			case <-t.C:
				if r.changed() {
					if nb, err := loadBundle(r.cfg); err == nil {
						r.snap()
						r.apply(nb)
					} else {
						log.Printf("mtls: reload failed: %v", err)
					}
				}
			case <-r.stopCh:
				t.Stop()
				return
			}
		}
	}()
}

func (r *Reloader) Stop() { close(r.stopCh) }

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

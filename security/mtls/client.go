package mtls

import (
	"crypto/tls"
	"errors"
	"log"
	"os"
	"strings"
	"time"
)

// TLSConfigClient constructs a client-side *tls.Config for mTLS.
func TLSConfigClient(c Config) (*tls.Config, *Reloader, error) {
	// guard: allow insecure only in dev-like envs
	if c.InsecureSkipVerify {
		if !envIsDev() {
			return nil, nil, errors.New("mtls: InsecureSkipVerify=true is forbidden outside DEV (set ENV=dev or APP_ENV=dev for local testing)")
		}
		log.Printf("mtls: *** WARNING *** InsecureSkipVerify=true — server cert verification is DISABLED (DEV only)")
	}

	b, err := loadBundle(c)
	if err != nil {
		return nil, nil, err
	}

	tlsConf := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		RootCAs:            b.rootPool,
		Certificates:       []tls.Certificate{b.cert},
		InsecureSkipVerify: c.InsecureSkipVerify, // DEV only (guarded above)
	}

	// When verification is ON (normal), set ServerName for SAN check if provided.
	if c.ServerName != "" {
		tlsConf.ServerName = c.ServerName
	}

	var r *Reloader
	if c.ReloadInterval > 0 {
		r = NewReloader(c, func(nb *bundle) {
			tlsConf.RootCAs = nb.rootPool
			tlsConf.Certificates = []tls.Certificate{nb.cert}
		})
		r.Start(time.NewTicker(c.ReloadInterval))
	}

	return tlsConf, r, nil
}

// envIsDev reports whether process runs in a dev-like environment.
// We check common variables: ENV / APP_ENV / GO_ENV.
func envIsDev() bool {
	is := func(v string) bool {
		v = strings.TrimSpace(strings.ToLower(v))
		return v == "dev" || v == "development" || v == "local"
	}
	return is(os.Getenv("ENV")) || is(os.Getenv("APP_ENV")) || is(os.Getenv("GO_ENV"))
}

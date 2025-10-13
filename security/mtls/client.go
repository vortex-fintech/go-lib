package mtls

import (
	"crypto/tls"
	"time"
)

// TLSConfigClient constructs a client-side *tls.Config for mTLS.
func TLSConfigClient(c Config) (*tls.Config, *Reloader, error) {
	b, err := loadBundle(c)
	if err != nil {
		return nil, nil, err
	}

	tlsConf := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		RootCAs:      b.rootPool,
		Certificates: []tls.Certificate{b.cert},
	}

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

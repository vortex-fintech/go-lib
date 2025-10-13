package mtls

import (
	"crypto/tls"
	"time"
)

// TLSConfigServer constructs a hardened server-side *tls.Config for mTLS.
func TLSConfigServer(c Config) (*tls.Config, *Reloader, error) {
	b, err := loadBundle(c)
	if err != nil {
		return nil, nil, err
	}

	tlsConf := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
		ClientAuth:               tls.RequireAndVerifyClientCert,
		ClientCAs:                b.rootPool,
		Certificates:             []tls.Certificate{b.cert},
		// Disable session tickets to reduce key material reuse in internal mesh.
		SessionTicketsDisabled: true,
	}

	// Curve and cipher preferences are mostly automatic in modern Go; keep defaults.

	// Optional reloader.
	var r *Reloader
	if c.ReloadInterval > 0 {
		r = NewReloader(c, func(nb *bundle) {
			// swap references in place
			tlsConf.ClientCAs = nb.rootPool
			tlsConf.Certificates = []tls.Certificate{nb.cert}
		})
		r.Start(time.NewTicker(c.ReloadInterval))
	}

	return tlsConf, r, nil
}

package mtls

import (
	"crypto/tls"
	"sync/atomic"
	"time"
)

// TLSConfigServer constructs a hardened server-side *tls.Config for mTLS.
func TLSConfigServer(c Config) (*tls.Config, *Reloader, error) {
	b, err := loadBundle(c)
	if err != nil {
		return nil, nil, err
	}

	state := &atomic.Pointer[bundle]{}
	state.Store(b)

	tlsConf := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
		ClientAuth:               tls.RequireAndVerifyClientCert,
		ClientCAs:                b.rootPool,
		Certificates:             []tls.Certificate{b.cert},
		// Disable session tickets to reduce key material reuse in internal mesh.
		SessionTicketsDisabled: true,
	}
	tlsConf.GetConfigForClient = func(*tls.ClientHelloInfo) (*tls.Config, error) {
		current := state.Load()
		conf := tlsConf.Clone()
		conf.ClientCAs = current.rootPool
		conf.Certificates = []tls.Certificate{current.cert}
		conf.GetConfigForClient = nil
		return conf, nil
	}

	// Curve and cipher preferences are mostly automatic in modern Go; keep defaults.

	// Optional reloader.
	var r *Reloader
	if c.ReloadInterval > 0 {
		r = NewReloader(c, func(nb *bundle) {
			state.Store(nb)
		})
		r.Start(time.NewTicker(c.ReloadInterval))
	}

	return tlsConf, r, nil
}

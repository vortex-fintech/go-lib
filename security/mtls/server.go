package mtls

import (
	"crypto/tls"
	"sync/atomic"
	"time"
)

func TLSConfigServer(c Config) (*tls.Config, *Reloader, error) {
	b, err := loadBundle(c)
	if err != nil {
		return nil, nil, err
	}

	state := &atomic.Pointer[bundle]{}
	state.Store(b)

	tlsConf := &tls.Config{
		MinVersion:             tls.VersionTLS13,
		ClientAuth:             tls.RequireAndVerifyClientCert,
		ClientCAs:              b.rootPool,
		Certificates:           []tls.Certificate{b.cert},
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

	var r *Reloader
	if c.ReloadInterval > 0 {
		r = NewReloader(c, func(nb *bundle) {
			state.Store(nb)
		})
		r.Start(time.NewTicker(c.ReloadInterval))
	}

	return tlsConf, r, nil
}

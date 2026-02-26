package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"
)

func TLSConfigClient(c Config) (*tls.Config, *Reloader, error) {
	b, err := loadBundle(c)
	if err != nil {
		return nil, nil, err
	}

	if c.ServerName == "" {
		slog.Warn("mtls: ServerName is empty, hostname verification disabled")
	}

	state := &atomic.Pointer[bundle]{}
	state.Store(b)

	tlsConf := &tls.Config{
		MinVersion: tls.VersionTLS13,
		GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			current := state.Load()
			return &current.cert, nil
		},
	}

	if c.ServerName != "" {
		tlsConf.ServerName = c.ServerName
	}

	tlsConf.InsecureSkipVerify = true
	tlsConf.VerifyConnection = func(cs tls.ConnectionState) error {
		if len(cs.PeerCertificates) == 0 {
			return errors.New("mtls: missing peer certificate")
		}

		intermediates := x509.NewCertPool()
		for _, cert := range cs.PeerCertificates[1:] {
			intermediates.AddCert(cert)
		}

		opts := x509.VerifyOptions{
			Roots:         state.Load().rootPool,
			Intermediates: intermediates,
			CurrentTime:   time.Now(),
		}
		if c.ServerName != "" {
			opts.DNSName = c.ServerName
		}

		_, err := cs.PeerCertificates[0].Verify(opts)
		return err
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

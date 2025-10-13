package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// bundle keeps parsed TLS artifacts.
type bundle struct {
	cert     tls.Certificate
	rootPool *x509.CertPool
	certStat fs.FileInfo
	keyStat  fs.FileInfo
	caStat   fs.FileInfo
}

// loadBundle reads CA, cert, and key from disk and returns parsed materials.
func loadBundle(c Config) (*bundle, error) {
	if c.CACertPath == "" || c.CertPath == "" || c.KeyPath == "" {
		return nil, errors.New("mtls: CACertPath, CertPath, and KeyPath are required")
	}

	caPEM, err := os.ReadFile(c.CACertPath)
	if err != nil {
		return nil, fmt.Errorf("mtls: read CA: %w", err)
	}
	roots := x509.NewCertPool()
	if ok := roots.AppendCertsFromPEM(caPEM); !ok {
		return nil, errors.New("mtls: failed to parse CA PEM")
	}

	crt, err := tls.LoadX509KeyPair(c.CertPath, c.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("mtls: load key pair: %w", err)
	}

	caStat, _ := os.Stat(c.CACertPath)
	certStat, _ := os.Stat(c.CertPath)
	keyStat, _ := os.Stat(c.KeyPath)

	return &bundle{
		cert:     crt,
		rootPool: roots,
		certStat: certStat,
		keyStat:  keyStat,
		caStat:   caStat,
	}, nil
}

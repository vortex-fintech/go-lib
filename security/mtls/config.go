package mtls

import "time"

// Config defines where to load cert materials and which SNI to expect.
type Config struct {
	CACertPath string // path to CA PEM (can contain multiple CAs)
	CertPath   string // path to leaf certificate PEM
	KeyPath    string // path to leaf private key PEM

	// Client side only: expected server name for SNI and hostname verification.
	ServerName string

	// Optional: enable periodic reload of certs without process restart.
	// If zero, reloading is disabled.
	ReloadInterval time.Duration
}

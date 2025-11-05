package tlsutil

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
)

// X5tS256FromTLSConfig возвращает x5t#S256 из tls.Config.Certificates[0] (DER).
func X5tS256FromTLSConfig(cfg *tls.Config) (string, error) {
	if cfg == nil {
		return "", errors.New("tlsutil: nil tls.Config")
	}
	var leaf *x509.Certificate
	if len(cfg.Certificates) > 0 {
		c := cfg.Certificates[0]
		if c.Leaf != nil {
			leaf = c.Leaf
		} else if len(c.Certificate) > 0 {
			var err error
			leaf, err = x509.ParseCertificate(c.Certificate[0])
			if err != nil {
				return "", err
			}
		}
	}
	if leaf == nil {
		return "", errors.New("tlsutil: leaf certificate not found")
	}
	sum := sha256.Sum256(leaf.Raw)
	return base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

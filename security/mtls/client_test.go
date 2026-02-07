package mtls

import (
	"os"
	"testing"
)

func TestTLSConfigClient_OK(t *testing.T) {
	tc := createTempCerts(t)
	defer os.RemoveAll(tc.Dir)

	conf, r, err := TLSConfigClient(Config{
		CACertPath:     tc.CAPath,
		CertPath:       tc.ClientCert,
		KeyPath:        tc.ClientKey,
		ReloadInterval: 0,
	})
	if err != nil {
		t.Fatalf("TLSConfigClient: %v", err)
	}
	if r != nil {
		t.Fatalf("reloader should be nil when ReloadInterval=0")
	}
	if conf == nil {
		t.Fatalf("tls.Config is nil")
	}
	if len(conf.Certificates) == 0 && conf.GetClientCertificate == nil {
		t.Fatalf("client certificate source is missing")
	}
	if conf.RootCAs == nil && conf.VerifyConnection == nil {
		t.Fatalf("server trust source is missing")
	}
	if conf.GetClientCertificate == nil {
		t.Fatalf("GetClientCertificate callback is nil")
	}
	if conf.VerifyConnection == nil {
		t.Fatalf("VerifyConnection callback is nil")
	}
}

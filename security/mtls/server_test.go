package mtls

import (
	"os"
	"testing"
)

func TestTLSConfigServer_OK(t *testing.T) {
	tc := createTempCerts(t)
	defer os.RemoveAll(tc.Dir)

	conf, r, err := TLSConfigServer(Config{
		CACertPath:     tc.CAPath,
		CertPath:       tc.ServerCert,
		KeyPath:        tc.ServerKey,
		ReloadInterval: 0,
	})
	if err != nil {
		t.Fatalf("TLSConfigServer: %v", err)
	}
	if r != nil {
		t.Fatalf("reloader should be nil when ReloadInterval=0")
	}
	if conf == nil {
		t.Fatalf("tls.Config is nil")
	}
	if conf.ClientAuth == 0 {
		t.Fatalf("ClientAuth should be set (RequireAndVerifyClientCert expected)")
	}
	if len(conf.Certificates) == 0 {
		t.Fatalf("server certificate is missing")
	}
	if conf.ClientCAs == nil {
		t.Fatalf("ClientCAs is nil")
	}
	if conf.GetConfigForClient == nil {
		t.Fatalf("GetConfigForClient callback is nil")
	}
}

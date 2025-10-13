package mtls

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBundle_OK(t *testing.T) {
	tc := createTempCerts(t)
	defer os.RemoveAll(tc.Dir)

	b, err := loadBundle(Config{
		CACertPath: tc.CAPath,
		CertPath:   tc.ServerCert,
		KeyPath:    tc.ServerKey,
	})
	if err != nil {
		t.Fatalf("loadBundle: %v", err)
	}
	if b.rootPool == nil || len(b.cert.Certificate) == 0 {
		t.Fatalf("bundle incomplete")
	}
}

func TestLoadBundle_Errors(t *testing.T) {
	tc := createTempCerts(t)
	defer os.RemoveAll(tc.Dir)

	// Не существует CA
	if _, err := loadBundle(Config{
		CACertPath: filepath.Join(tc.Dir, "no_ca.pem"),
		CertPath:   tc.ServerCert,
		KeyPath:    tc.ServerKey,
	}); err == nil {
		t.Fatalf("expected error for missing CA")
	}

	// Битый cert/key
	badCert := filepath.Join(tc.Dir, "bad.pem")
	if err := os.WriteFile(badCert, []byte("not a pem"), 0o600); err != nil {
		t.Fatalf("write bad cert: %v", err)
	}
	if _, err := loadBundle(Config{
		CACertPath: tc.CAPath,
		CertPath:   badCert,
		KeyPath:    tc.ServerKey,
	}); err == nil {
		t.Fatalf("expected error for bad cert")
	}
}

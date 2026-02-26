package dial_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vortex-fintech/go-lib/security/mtls"
	"github.com/vortex-fintech/go-lib/transport/grpc/dial"
	gbackoff "google.golang.org/grpc/backoff"
)

type testCerts struct {
	CAPath     string
	ClientCert string
	ClientKey  string
	Dir        string
}

func createTempCerts(t *testing.T) testCerts {
	t.Helper()

	dir, err := os.MkdirTemp("", "dial-test-*")
	if err != nil {
		t.Fatalf("mkdtemp: %v", err)
	}

	caKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	caTpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "Test Root CA",
			Organization: []string{"Vortex"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	caDER, _ := x509.CreateCertificate(rand.Reader, caTpl, caTpl, &caKey.PublicKey, caKey)

	writePEM := func(path, typ string, der []byte) {
		f, err := os.Create(path)
		if err != nil {
			t.Fatalf("create %s: %v", path, err)
		}
		defer f.Close()
		if err := pem.Encode(f, &pem.Block{Type: typ, Bytes: der}); err != nil {
			t.Fatalf("pem encode %s: %v", path, err)
		}
	}

	caPath := filepath.Join(dir, "ca.pem")
	writePEM(caPath, "CERTIFICATE", caDER)

	cliKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	cliTpl := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject: pkix.Name{
			Organization: []string{"Vortex"},
		},
		NotBefore:   time.Now().Add(-time.Hour),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		DNSNames:    []string{"client.test.internal"},
	}
	cliDER, _ := x509.CreateCertificate(rand.Reader, cliTpl, caTpl, &cliKey.PublicKey, caKey)

	cliCrt := filepath.Join(dir, "client.pem")
	writePEM(cliCrt, "CERTIFICATE", cliDER)

	cliKeyPath := filepath.Join(dir, "client.key")
	{
		f, err := os.Create(cliKeyPath)
		if err != nil {
			t.Fatalf("create %s: %v", cliKeyPath, err)
		}
		defer f.Close()
		if err := pem.Encode(f, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(cliKey)}); err != nil {
			t.Fatalf("pem encode key: %v", err)
		}
	}

	return testCerts{
		CAPath:     caPath,
		ClientCert: cliCrt,
		ClientKey:  cliKeyPath,
		Dir:        dir,
	}
}

func TestNewClient_ValidConfig(t *testing.T) {
	t.Parallel()

	certs := createTempCerts(t)
	defer os.RemoveAll(certs.Dir)

	opt := dial.Options{
		MTLS: mtls.Config{
			CACertPath: certs.CAPath,
			CertPath:   certs.ClientCert,
			KeyPath:    certs.ClientKey,
			ServerName: "server.test.internal",
		},
	}

	conn, err := dial.NewClient(context.Background(), "passthrough:///localhost:0", opt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer conn.Close()
}

func TestNewClient_MissingCA(t *testing.T) {
	t.Parallel()

	certs := createTempCerts(t)
	defer os.RemoveAll(certs.Dir)

	opt := dial.Options{
		MTLS: mtls.Config{
			CACertPath: "/nonexistent/ca.pem",
			CertPath:   certs.ClientCert,
			KeyPath:    certs.ClientKey,
			ServerName: "server.test.internal",
		},
	}

	_, err := dial.NewClient(context.Background(), "passthrough:///localhost:0", opt)
	if err == nil {
		t.Fatalf("expected error for missing CA")
	}
}

func TestNewClient_MissingCert(t *testing.T) {
	t.Parallel()

	certs := createTempCerts(t)
	defer os.RemoveAll(certs.Dir)

	opt := dial.Options{
		MTLS: mtls.Config{
			CACertPath: certs.CAPath,
			CertPath:   "/nonexistent/client.pem",
			KeyPath:    certs.ClientKey,
			ServerName: "server.test.internal",
		},
	}

	_, err := dial.NewClient(context.Background(), "passthrough:///localhost:0", opt)
	if err == nil {
		t.Fatalf("expected error for missing cert")
	}
}

func TestNewClient_MissingKey(t *testing.T) {
	t.Parallel()

	certs := createTempCerts(t)
	defer os.RemoveAll(certs.Dir)

	opt := dial.Options{
		MTLS: mtls.Config{
			CACertPath: certs.CAPath,
			CertPath:   certs.ClientCert,
			KeyPath:    "/nonexistent/client.key",
			ServerName: "server.test.internal",
		},
	}

	_, err := dial.NewClient(context.Background(), "passthrough:///localhost:0", opt)
	if err == nil {
		t.Fatalf("expected error for missing key")
	}
}

func TestNewClient_CustomBackoff(t *testing.T) {
	t.Parallel()

	certs := createTempCerts(t)
	defer os.RemoveAll(certs.Dir)

	customBackoff := gbackoff.Config{
		BaseDelay:  50 * time.Millisecond,
		Multiplier: 2.0,
		Jitter:     0.1,
		MaxDelay:   5 * time.Second,
	}

	opt := dial.Options{
		MTLS: mtls.Config{
			CACertPath: certs.CAPath,
			CertPath:   certs.ClientCert,
			KeyPath:    certs.ClientKey,
			ServerName: "server.test.internal",
		},
		Backoff: customBackoff,
	}

	conn, err := dial.NewClient(context.Background(), "passthrough:///localhost:0", opt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer conn.Close()
}

func TestNewClient_CustomMessageSize(t *testing.T) {
	t.Parallel()

	certs := createTempCerts(t)
	defer os.RemoveAll(certs.Dir)

	opt := dial.Options{
		MTLS: mtls.Config{
			CACertPath: certs.CAPath,
			CertPath:   certs.ClientCert,
			KeyPath:    certs.ClientKey,
			ServerName: "server.test.internal",
		},
		MaxRecvMsgSize: 32 << 20,
		MaxSendMsgSize: 32 << 20,
	}

	conn, err := dial.NewClient(context.Background(), "passthrough:///localhost:0", opt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer conn.Close()
}

func TestNewClient_WithWindowSizes(t *testing.T) {
	t.Parallel()

	certs := createTempCerts(t)
	defer os.RemoveAll(certs.Dir)

	opt := dial.Options{
		MTLS: mtls.Config{
			CACertPath: certs.CAPath,
			CertPath:   certs.ClientCert,
			KeyPath:    certs.ClientKey,
			ServerName: "server.test.internal",
		},
		InitialWindow: 1 << 20,
		InitialConn:   2 << 20,
	}

	conn, err := dial.NewClient(context.Background(), "passthrough:///localhost:0", opt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer conn.Close()
}

func TestDial_BackwardCompatible(t *testing.T) {
	t.Parallel()

	certs := createTempCerts(t)
	defer os.RemoveAll(certs.Dir)

	opt := dial.Options{
		MTLS: mtls.Config{
			CACertPath: certs.CAPath,
			CertPath:   certs.ClientCert,
			KeyPath:    certs.ClientKey,
			ServerName: "server.test.internal",
		},
	}

	conn, err := dial.Dial("passthrough:///localhost:0", opt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer conn.Close()
}

func TestDefaultBackoff(t *testing.T) {
	t.Parallel()

	bc := dial.DefaultBackoff()
	if bc.BaseDelay != 100e6 {
		t.Fatalf("BaseDelay: got %v, want %v", bc.BaseDelay, 100e6)
	}
	if bc.Multiplier != 1.6 {
		t.Fatalf("Multiplier: got %v, want %v", bc.Multiplier, 1.6)
	}
	if bc.Jitter != 0.2 {
		t.Fatalf("Jitter: got %v, want %v", bc.Jitter, 0.2)
	}
	if bc.MaxDelay != 2e9 {
		t.Fatalf("MaxDelay: got %v, want %v", bc.MaxDelay, 2e9)
	}
}

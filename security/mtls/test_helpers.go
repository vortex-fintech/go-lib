package mtls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

type testCerts struct {
	CAPath     string
	ServerCert string
	ServerKey  string
	ClientCert string
	ClientKey  string
	Dir        string
}

// createTempCerts генерирует временной CA и подписанные им server/client certs.
// Возвращает пути к PEM-файлам и директорию, которую можно удалить после теста.
func createTempCerts(t interface {
	Helper()
	Fatalf(string, ...any)
}) testCerts {
	t.Helper()

	dir, err := os.MkdirTemp("", "mtls-test-*")
	if err != nil {
		t.Fatalf("mkdtemp: %v", err)
	}

	// --- CA ---
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

	// helper для записи PEM
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

	// write CA cert
	caPath := filepath.Join(dir, "ca.pem")
	writePEM(caPath, "CERTIFICATE", caDER)

	// --- Server cert ---
	srvKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	srvTpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Vortex"},
		},
		NotBefore:   time.Now().Add(-time.Hour),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		// Use SAN for hostname verification (modern clients ignore CN)
		DNSNames: []string{"server.test.internal"},
	}
	srvDER, _ := x509.CreateCertificate(rand.Reader, srvTpl, caTpl, &srvKey.PublicKey, caKey)

	srvCrt := filepath.Join(dir, "server.pem")
	writePEM(srvCrt, "CERTIFICATE", srvDER)

	srvKeyPath := filepath.Join(dir, "server.key")
	{
		f, err := os.Create(srvKeyPath)
		if err != nil {
			t.Fatalf("create %s: %v", srvKeyPath, err)
		}
		defer f.Close()
		if err := pem.Encode(f, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(srvKey)}); err != nil {
			t.Fatalf("pem encode key: %v", err)
		}
	}

	// --- Client cert ---
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
		// Not strictly required for mTLS client auth, but keep SAN for completeness
		DNSNames: []string{"client.test.internal"},
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
		ServerCert: srvCrt,
		ServerKey:  srvKeyPath,
		ClientCert: cliCrt,
		ClientKey:  cliKeyPath,
		Dir:        dir,
	}
}

package creds_test

import (
	"crypto/tls"
	"crypto/x509"
	"testing"

	"github.com/vortex-fintech/go-lib/transport/grpc/creds"
)

func TestServerTransportCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tlsConf *tls.Config
		opt     creds.ServerOptions
		wantErr error
	}{
		{
			name:    "nil config",
			tlsConf: nil,
			opt:     creds.ServerOptions{},
			wantErr: creds.ErrNilTLSConfig,
		},
		{
			name:    "missing certificate",
			tlsConf: &tls.Config{},
			opt:     creds.ServerOptions{},
			wantErr: creds.ErrMissingCert,
		},
		{
			name: "missing client ca with require client cert",
			tlsConf: &tls.Config{
				Certificates: []tls.Certificate{{}},
				ClientAuth:   tls.RequireAndVerifyClientCert,
			},
			opt:     creds.ServerOptions{},
			wantErr: creds.ErrMissingRootCA,
		},
		{
			name: "valid config with no client auth",
			tlsConf: &tls.Config{
				Certificates: []tls.Certificate{{}},
				ClientAuth:   tls.NoClientCert,
			},
			opt:     creds.ServerOptions{},
			wantErr: nil,
		},
		{
			name: "valid config with client ca",
			tlsConf: &tls.Config{
				Certificates: []tls.Certificate{{}},
				ClientAuth:   tls.RequireAndVerifyClientCert,
				ClientCAs:    x509.NewCertPool(),
			},
			opt:     creds.ServerOptions{},
			wantErr: nil,
		},
		{
			name: "skip root ca validation",
			tlsConf: &tls.Config{
				Certificates: []tls.Certificate{{}},
				ClientAuth:   tls.RequireAndVerifyClientCert,
			},
			opt:     creds.ServerOptions{SkipRootCAValidation: true},
			wantErr: nil,
		},
		{
			name: "valid config with GetCertificate",
			tlsConf: &tls.Config{
				GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
					return &tls.Certificate{}, nil
				},
				ClientAuth: tls.NoClientCert,
			},
			opt:     creds.ServerOptions{},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := creds.ServerTransportCredentials(tt.tlsConf, tt.opt)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				if got != nil {
					t.Fatalf("expected nil credentials on error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil {
				t.Fatalf("expected non-nil credentials")
			}
		})
	}
}

func TestClientTransportCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tlsConf *tls.Config
		opt     creds.ClientOptions
		wantErr error
	}{
		{
			name:    "nil config",
			tlsConf: nil,
			opt:     creds.ClientOptions{},
			wantErr: creds.ErrNilTLSConfig,
		},
		{
			name:    "missing root ca",
			tlsConf: &tls.Config{},
			opt:     creds.ClientOptions{},
			wantErr: creds.ErrMissingRootCA,
		},
		{
			name:    "valid config with root ca",
			tlsConf: &tls.Config{RootCAs: x509.NewCertPool()},
			opt:     creds.ClientOptions{},
			wantErr: nil,
		},
		{
			name:    "insecure skip verify",
			tlsConf: &tls.Config{},
			opt:     creds.ClientOptions{InsecureSkipVerify: true},
			wantErr: nil,
		},
		{
			name:    "skip root ca validation",
			tlsConf: &tls.Config{},
			opt:     creds.ClientOptions{SkipRootCAValidation: true},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := creds.ClientTransportCredentials(tt.tlsConf, tt.opt)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				if got != nil {
					t.Fatalf("expected nil credentials on error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil {
				t.Fatalf("expected non-nil credentials")
			}
		})
	}
}

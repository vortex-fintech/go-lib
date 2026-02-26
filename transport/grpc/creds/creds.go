package creds

import (
	"crypto/tls"
	"errors"

	"google.golang.org/grpc/credentials"
)

var (
	ErrNilTLSConfig  = errors.New("tls.Config cannot be nil")
	ErrMissingCert   = errors.New("tls.Config must have at least one certificate")
	ErrMissingRootCA = errors.New("tls.Config must have RootCAs for client or ClientCAs for server")
)

type ServerOptions struct {
	SkipRootCAValidation bool
}

type ClientOptions struct {
	SkipRootCAValidation bool
	InsecureSkipVerify   bool
}

func ServerTransportCredentials(tlsConf *tls.Config, opt ServerOptions) (credentials.TransportCredentials, error) {
	if tlsConf == nil {
		return nil, ErrNilTLSConfig
	}

	if len(tlsConf.Certificates) == 0 && tlsConf.GetCertificate == nil {
		return nil, ErrMissingCert
	}

	if !opt.SkipRootCAValidation && tlsConf.ClientCAs == nil && tlsConf.ClientAuth != tls.NoClientCert {
		return nil, ErrMissingRootCA
	}

	return credentials.NewTLS(tlsConf), nil
}

func ClientTransportCredentials(tlsConf *tls.Config, opt ClientOptions) (credentials.TransportCredentials, error) {
	if tlsConf == nil {
		return nil, ErrNilTLSConfig
	}

	if !opt.SkipRootCAValidation && !opt.InsecureSkipVerify && tlsConf.RootCAs == nil {
		return nil, ErrMissingRootCA
	}

	if opt.InsecureSkipVerify {
		tlsConfCopy := tlsConf.Clone()
		tlsConfCopy.InsecureSkipVerify = true
		return credentials.NewTLS(tlsConfCopy), nil
	}

	return credentials.NewTLS(tlsConf), nil
}

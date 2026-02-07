package creds

import (
	"crypto/tls"

	"google.golang.org/grpc/credentials"
)

// ServerTransportCredentials wraps a *tls.Config for gRPC servers.
func ServerTransportCredentials(tlsConf *tls.Config) credentials.TransportCredentials {
	return credentials.NewTLS(tlsConf)
}

// ClientTransportCredentials wraps a *tls.Config for gRPC clients.
func ClientTransportCredentials(tlsConf *tls.Config) credentials.TransportCredentials {
	return credentials.NewTLS(tlsConf)
}

# gRPC TLS Credentials

Production-ready TLS credentials for gRPC servers and clients with validation.

## Where to use it

- gRPC servers requiring TLS/mTLS
- gRPC clients connecting to TLS endpoints
- Internal service-to-service communication with mutual TLS

## Basic usage

### Server

```go
tlsConf := &tls.Config{
    Certificates: []tls.Certificate{cert},
    ClientAuth:   tls.RequireAndVerifyClientCert,
    ClientCAs:    caPool,
}

creds, err := creds.ServerTransportCredentials(tlsConf, creds.ServerOptions{})
if err != nil {
    log.Fatal(err)
}

server := grpc.NewServer(grpc.Creds(creds))
```

### Client

```go
tlsConf := &tls.Config{
    RootCAs: caPool,
}

creds, err := creds.ClientTransportCredentials(tlsConf, creds.ClientOptions{})
if err != nil {
    log.Fatal(err)
}

conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(creds))
```

## ServerOptions

| Option | Default | Description |
|--------|---------|-------------|
| `SkipRootCAValidation` | false | Skip ClientCAs validation for servers |

## ClientOptions

| Option | Default | Description |
|--------|---------|-------------|
| `SkipRootCAValidation` | false | Skip RootCAs validation |
| `InsecureSkipVerify` | false | Skip server certificate verification (dev only) |

## Validation errors

| Error | Condition |
|-------|-----------|
| `ErrNilTLSConfig` | tls.Config is nil |
| `ErrMissingCert` | Server config has no certificates |
| `ErrMissingRootCA` | Client config has no RootCAs, or server config has no ClientCAs when mTLS enabled |

## With mTLS

```go
import "github.com/vortex-fintech/go-lib/security/mtls"

tlsConf, _, err := mtls.TLSConfigClient(mtls.Config{
    CA:         "/path/to/ca.pem",
    ClientCert: "/path/to/client.pem",
    ClientKey:  "/path/to/client.key",
    ServerName: "service.internal",
})
if err != nil {
    log.Fatal(err)
}

creds, err := creds.ClientTransportCredentials(tlsConf, creds.ClientOptions{})
```

## Development mode

For local development with self-signed certificates:

```go
creds, err := creds.ClientTransportCredentials(&tls.Config{}, creds.ClientOptions{
    InsecureSkipVerify: true,
})
```

**Warning:** Never use `InsecureSkipVerify: true` in production.

## Production notes

- Always provide valid `tls.Config` with certificates
- Use `ClientAuth: tls.RequireAndVerifyClientCert` for mTLS
- Set `ClientCAs` / `RootCAs` to trusted CA pool
- Keep `InsecureSkipVerify` false in production
- Use `mtls` package for hot-reloadable TLS configuration

# mTLS Configuration

Mutual TLS configuration with hot certificate reload.

## Where to use it

- Service-to-service authentication in internal mesh
- Client certificate verification
- Zero-downtime certificate rotation

## Server-side mTLS

```go
cfg := mtls.Config{
    CACertPath:     "/certs/ca.pem",
    CertPath:       "/certs/server.pem",
    KeyPath:        "/certs/server-key.pem",
    ReloadInterval: time.Minute,
}

tlsConfig, reloader, err := mtls.TLSConfigServer(cfg)
if err != nil {
    log.Fatal(err)
}
defer reloader.Stop()

server := &http.Server{
    Addr:      ":8443",
    TLSConfig: tlsConfig,
}
server.ListenAndServeTLS("", "")
```

## Client-side mTLS

```go
cfg := mtls.Config{
    CACertPath:  "/certs/ca.pem",
    CertPath:    "/certs/client.pem",
    KeyPath:     "/certs/client-key.pem",
    ServerName:  "api.internal", // required for hostname verification
}

tlsConfig, reloader, err := mtls.TLSConfigClient(cfg)
if err != nil {
    log.Fatal(err)
}
defer reloader.Stop()

client := &http.Client{
    Transport: &http.Transport{TLSClientConfig: tlsConfig},
}
```

## Hot reload

Certificates reload automatically when files change:

```go
cfg := mtls.Config{
    CACertPath:     "/certs/ca.pem",
    CertPath:       "/certs/server.pem",
    KeyPath:        "/certs/server-key.pem",
    ReloadInterval: 30 * time.Second,
}

tlsConfig, reloader, _ := mtls.TLSConfigServer(cfg)
defer reloader.Stop()
```

### Custom reload logger

```go
reloader := mtls.NewReloaderWithLogger(cfg, applyFunc, func(ev mtls.ReloadEvent) {
    if ev.Err != nil {
        slog.Error("cert reload failed", "error", ev.Err)
    } else {
        slog.Info("certificates reloaded")
    }
})
```

### Disable reload

```go
cfg := mtls.Config{
    CACertPath: "/certs/ca.pem",
    CertPath:   "/certs/server.pem",
    KeyPath:    "/certs/server-key.pem",
    // ReloadInterval zero = disabled
}
```

## Security defaults

| Setting | Value |
|---------|-------|
| Min TLS version | 1.3 |
| Client auth | RequireAndVerifyClientCert |
| Session tickets | Disabled |
| Cipher suites | TLS 1.3 only (AES-128-GCM, AES-256-GCM, ChaCha20-Poly1305) |

## Config reference

| Field | Description |
|-------|-------------|
| `CACertPath` | Path to CA PEM (can contain multiple CAs) |
| `CertPath` | Path to leaf certificate PEM |
| `KeyPath` | Path to private key PEM |
| `ServerName` | Expected server name (client only, recommended) |
| `ReloadInterval` | Polling interval for cert changes |

## ServerName warning

If `ServerName` is empty on client side, hostname verification is disabled and a warning is logged. Always set `ServerName` to the expected server hostname for security.

## Production notes

- Store certificates in secure location (Vault, Kubernetes secrets)
- Use short-lived certificates (24-72 hours)
- Set `ReloadInterval` to 30-60 seconds
- Monitor reload events via custom logger
- Always set `ServerName` on client side
- Test rotation before production

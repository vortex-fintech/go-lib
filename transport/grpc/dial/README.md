# gRPC Dial

Production-ready gRPC client factory with mTLS, backoff, and connection tuning.

## Where to use it

- Service-to-service gRPC clients
- mTLS authenticated connections
- High-throughput services needing TCP window tuning

## Basic usage

```go
conn, err := dial.NewClient(ctx, "api.internal:50051", dial.Options{
    MTLS: mtls.Config{
        CACertPath: "/etc/certs/ca.pem",
        CertPath:   "/etc/certs/client.pem",
        KeyPath:    "/etc/certs/client.key",
        ServerName: "api.internal",
    },
})
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pb.NewServiceClient(conn)
```

## Options

| Option | Default | Description |
|--------|---------|-------------|
| `MTLS` | required | mTLS configuration (see `security/mtls`) |
| `Backoff` | `DefaultBackoff()` | Connection retry backoff |
| `InitialWindow` | none | Initial stream window size |
| `InitialConn` | none | Initial connection window size |
| `MaxRecvMsgSize` | 16MB | Max message size to receive |
| `MaxSendMsgSize` | 16MB | Max message size to send |

## Backoff configuration

```go
opt := dial.Options{
    MTLS: mtls.Config{...},
    Backoff: gbackoff.Config{
        BaseDelay:  100 * time.Millisecond,
        Multiplier: 1.6,
        Jitter:     0.2,
        MaxDelay:   2 * time.Second,
    },
}
```

Default backoff: 100ms base, 1.6x multiplier, 0.2 jitter, 2s max.

## Message size tuning

For services with large payloads:

```go
opt := dial.Options{
    MTLS:           mtls.Config{...},
    MaxRecvMsgSize: 64 << 20, // 64MB
    MaxSendMsgSize: 64 << 20,
}
```

## TCP window tuning

For high-throughput services:

```go
opt := dial.Options{
    MTLS:          mtls.Config{...},
    InitialWindow: 1 << 20, // 1MB stream window
    InitialConn:   2 << 20, // 2MB connection window
}
```

## Hot reload certificates

```go
opt := dial.Options{
    MTLS: mtls.Config{
        CACertPath:    "/etc/certs/ca.pem",
        CertPath:      "/etc/certs/client.pem",
        KeyPath:       "/etc/certs/client.key",
        ServerName:    "api.internal",
        ReloadInterval: 5 * time.Minute, // auto-reload certs
    },
}
```

## Backward compatibility

```go
// Dial is an alias for NewClient(context.Background(), ...)
conn, err := dial.Dial("api.internal:50051", opt)
```

## Production notes

- Always use mTLS for internal services
- Set `ServerName` for hostname verification
- Use `ReloadInterval` for zero-downtime cert rotation
- Tune message sizes based on your payload requirements
- Connection is non-blocking - first RPC may fail if server is down

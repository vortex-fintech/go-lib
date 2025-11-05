package dial

import (
	"time"

	"github.com/vortex-fintech/go-lib/grpc/creds" // твой пакет creds
	"github.com/vortex-fintech/go-lib/security/mtls"

	"google.golang.org/grpc"
	gbackoff "google.golang.org/grpc/backoff"
)

// Options — минимальный набор для клиента.
type Options struct {
	MTLS mtls.Config

	// gRPC conn-backoff (не про retry RPC, а про реконнект канала).
	Backoff        gbackoff.Config
	InitialWindow  int32
	InitialConnWin int32

	// Если true — grpc.Dial будет блокирующим до установления коннекта.
	Block bool
}

func DefaultBackoff() gbackoff.Config {
	return gbackoff.Config{
		BaseDelay:  100 * time.Millisecond,
		Multiplier: 1.6,
		Jitter:     0.2,
		MaxDelay:   2 * time.Second,
	}
}

func Dial(target string, opt Options) (*grpc.ClientConn, error) {
	tlsConf, _, err := mtls.TLSConfigClient(opt.MTLS)
	if err != nil {
		return nil, err
	}

	bc := opt.Backoff
	if bc.BaseDelay == 0 {
		bc = DefaultBackoff()
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds.ClientTransportCredentials(tlsConf)),
		grpc.WithConnectParams(grpc.ConnectParams{Backoff: bc, MinConnectTimeout: 3 * time.Second}),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(16<<20),
			grpc.MaxCallSendMsgSize(16<<20),
		),
	}
	if opt.InitialWindow > 0 {
		opts = append(opts, grpc.WithInitialWindowSize(opt.InitialWindow))
	}
	if opt.InitialConnWin > 0 {
		opts = append(opts, grpc.WithInitialConnWindowSize(opt.InitialConnWin))
	}
	if opt.Block {
		opts = append(opts, grpc.WithBlock())
	}

	return grpc.Dial(target, opts...)
}

package dial

import (
	"context"

	"github.com/vortex-fintech/go-lib/security/mtls"
	"github.com/vortex-fintech/go-lib/transport/grpc/creds"

	"google.golang.org/grpc"
	gbackoff "google.golang.org/grpc/backoff"
)

type Options struct {
	MTLS mtls.Config

	Backoff       gbackoff.Config
	InitialWindow int32
	InitialConn   int32

	MaxRecvMsgSize int
	MaxSendMsgSize int
}

func DefaultBackoff() gbackoff.Config {
	return gbackoff.Config{
		BaseDelay:  100e6,
		Multiplier: 1.6,
		Jitter:     0.2,
		MaxDelay:   2e9,
	}
}

func NewClient(ctx context.Context, target string, opt Options) (*grpc.ClientConn, error) {
	tlsConf, _, err := mtls.TLSConfigClient(opt.MTLS)
	if err != nil {
		return nil, err
	}

	cred, err := creds.ClientTransportCredentials(tlsConf, creds.ClientOptions{
		SkipRootCAValidation: true,
	})
	if err != nil {
		return nil, err
	}

	bc := opt.Backoff
	if bc.BaseDelay == 0 {
		bc = DefaultBackoff()
	}

	maxRecv := opt.MaxRecvMsgSize
	if maxRecv == 0 {
		maxRecv = 16 << 20
	}
	maxSend := opt.MaxSendMsgSize
	if maxSend == 0 {
		maxSend = 16 << 20
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(cred),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff:           bc,
			MinConnectTimeout: 3e9,
		}),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxRecv),
			grpc.MaxCallSendMsgSize(maxSend),
		),
	}
	if opt.InitialWindow > 0 {
		opts = append(opts, grpc.WithInitialWindowSize(opt.InitialWindow))
	}
	if opt.InitialConn > 0 {
		opts = append(opts, grpc.WithInitialConnWindowSize(opt.InitialConn))
	}

	return grpc.NewClient(target, opts...)
}

func Dial(target string, opt Options) (*grpc.ClientConn, error) {
	return NewClient(context.Background(), target, opt)
}

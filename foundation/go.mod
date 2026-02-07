module github.com/vortex-fintech/go-lib/foundation

go 1.25

toolchain go1.25.1

require (
	github.com/cenkalti/backoff/v5 v5.0.2
	github.com/go-playground/validator/v10 v10.27.0
	github.com/google/uuid v1.6.0
	github.com/stretchr/testify v1.11.1
	go.uber.org/zap v1.27.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250324211829-b45e905df463
	google.golang.org/grpc v1.73.0
)

require google.golang.org/protobuf v1.36.8 // indirect

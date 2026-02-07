module github.com/vortex-fintech/go-lib/transport

go 1.25

toolchain go1.25.1

require (
	github.com/google/uuid v1.6.0
	github.com/vortex-fintech/go-lib/foundation v0.0.0
	github.com/vortex-fintech/go-lib/security v0.0.0
	google.golang.org/grpc v1.73.0
)

replace github.com/vortex-fintech/go-lib/foundation => ../foundation

replace github.com/vortex-fintech/go-lib/security => ../security

module github.com/vortex-fintech/go-lib/transport

go 1.25

toolchain go1.25.1

require (
	github.com/google/uuid v1.6.0
	github.com/vortex-fintech/go-lib/foundation v0.0.0
	github.com/vortex-fintech/go-lib/security v0.0.0
	google.golang.org/grpc v1.78.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.27.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/crypto v0.44.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda // indirect
)

replace github.com/vortex-fintech/go-lib/foundation => ../foundation

replace github.com/vortex-fintech/go-lib/security => ../security

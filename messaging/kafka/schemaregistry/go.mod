module github.com/vortex-fintech/go-lib/messaging/kafka/schemaregistry

go 1.25

toolchain go1.25.1

require google.golang.org/protobuf v1.36.11

require github.com/twmb/franz-go/pkg/sr v1.6.0

replace github.com/vortex-fintech/go-lib/messaging/kafka/schemaregistry => ../

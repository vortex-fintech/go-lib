module github.com/vortex-fintech/go-lib/messaging/kafka/franzgo

go 1.25

toolchain go1.25.1

require github.com/twmb/franz-go v1.16.0

require (
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.19 // indirect
	github.com/twmb/franz-go/pkg/kmsg v1.7.0 // indirect
	golang.org/x/crypto v0.44.0 // indirect
)

replace github.com/vortex-fintech/go-lib/messaging/kafka/franzgo => ../

# messaging

Kafka messaging utilities for Vortex services.

## Packages

| Package | Description |
|---------|-------------|
| [franzgo](./kafka/franzgo) | Kafka client wrapper based on franz-go |
| [schemaregistry](./kafka/schemaregistry) | Schema Registry client with Protobuf support |

## Dependencies

- [franz-go](https://github.com/twmb/franz-go) - Apache Kafka client
- [protobuf](https://protobuf.dev/) - Protocol Buffers

## Architecture

```
messaging/
└── kafka/
    ├── franzgo/          # Client, Producer, Consumer
    └── schemaregistry/  # Schema Registry client, Serializer, Deserializer
```

## Quick Start

```go
import (
    "context"
    "log"
    
    "github.com/vortex-fintech/go-lib/messaging/kafka/franzgo"
    "github.com/vortex-fintech/go-lib/messaging/kafka/schemaregistry"
    paymentv1 "github.com/vortex-fintech/go-lib/gen/go/payment/v1"
)

type PaymentPublisher struct {
    client     *franzgo.Client
    producer  *franzgo.Producer
    serializer *schemaregistry.ProtoSerializer
}

func NewPaymentPublisher(kafkaBrokers []string, schemaRegistryURL string) (*PaymentPublisher, error) {
    client, err := franzgo.NewClient(franzgo.Config{
        SeedBrokers: kafkaBrokers,
        ClientID:    "payment-service",
    })
    if err != nil {
        return nil, err
    }

    srClient, err := schemaregistry.NewClient(schemaregistry.Config{
        URL:     schemaRegistryURL,
        Timeout: 5,
    })
    if err != nil {
        client.Close()
        return nil, err
    }

    return &PaymentPublisher{
        client:     client,
        producer:  franzgo.NewProducer(client, "payment-events"),
        serializer: schemaregistry.NewProtoSerializer(srClient),
    }, nil
}

func (p *PaymentPublisher) Publish(ctx context.Context, event *paymentv1.PaymentCompleted) error {
    payload, _, err := p.serializer.SerializeWithSchema(
        "payment-events-value",
        paymentSchemaProto,
        event,
    )
    if err != nil {
        return err
    }

    return p.producer.Produce(ctx, []byte(event.PaymentId), payload)
}

func (p *PaymentPublisher) Close() {
    p.client.Close()
}
```

## Configuration

### Kafka (Environment Variables)

```env
KAFKA_BROKERS=kafka-1:9092,kafka-2:9092,kafka-3:9092
KAFKA_CLIENT_ID=payment-service
KAFKA_CONSUMER_GROUP=payment-consumers
```

### Schema Registry
```env
SCHEMA_REGISTRY_URL=http://schema-registry:8081
SCHEMA_REGISTRY_TIMEOUT=5
```

## Testing

```bash
# Run all tests from repository root
go test ./messaging/kafka/franzgo/...
go test ./messaging/kafka/schemaregistry/...

# Run with race detection in Docker (from repository root)
docker run --rm -v "C:/Vortex Services/go-lib:/work" golang:1.25.7 \
    sh -c 'cd /work && go test -race ./messaging/kafka/franzgo/... && go test -race ./messaging/kafka/schemaregistry/...'
```

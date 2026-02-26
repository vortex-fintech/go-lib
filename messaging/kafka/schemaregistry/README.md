# schemaregistry

Schema Registry client and Protobuf serializer/deserializer for Kafka.

## Features

- Schema Registry client with timeout support
- Protobuf serialization with schema caching
- Confluent wire format support
- Schema validation

## Installation

```go
go get github.com/vortex-fintech/go-lib/messaging/kafka/schemaregistry
```

## Quick Start

### Client

```go
import "github.com/vortex-fintech/go-lib/messaging/kafka/schemaregistry"

func main() {
    client, err := schemaregistry.NewClient(schemaregistry.Config{
        URL:     "http://localhost:8081",
        Timeout: 5,
    })
    if err != nil {
        panic(err)
    }

    subjects, err := client.GetAllSubjects()
    if err != nil {
        panic(err)
    }
}
```

### Serializer

```go
serializer := schemaregistry.NewProtoSerializer(client)

event := &paymentv1.PaymentCompleted{
    PaymentId:   "pay-123",
    Amount:      10000,
    Currency:    "USD",
    CompletedAt: timestamppb.Now(),
}

payload, schemaID, err := serializer.SerializeWithSchema(
    "payment-events-value",
    paymentSchemaProto,
    event,
)
if err != nil {
    panic(err)
}

producer.Produce(ctx, []byte("pay-123"), payload)
```

### Deserializer

```go
deserializer := schemaregistry.NewProtoDeserializer(client)

payload, schemaID, err := deserializer.Deserialize(msg.Value)
if err != nil {
    panic(err)
}

var event paymentv1.PaymentCompleted
if err := proto.Unmarshal(payload, &event); err != nil {
    panic(err)
}
```

## Wire Format

The serializer produces Confluent wire format:

```
+--------+--------+--------+--------+--------+--------+--------+
| Magic  |      Schema ID     |  Index |   Protobuf Payload    |
| 1 byte |      4 bytes       | 1+ var |      N bytes          |
+--------+--------+--------+--------+--------+--------+--------+
```

- Magic byte: `0x00`
- Schema ID: Big-endian 4-byte integer
- Index: Protobuf message index path derived from the concrete message descriptor
- Payload: Protobuf serialized data

## Client Methods

| Method | Description |
|--------|-------------|
| `GetLatestSchema(subject)` | Get latest schema for subject |
| `RegisterSchema(subject, schema)` | Register new schema |
| `RegisterSchemaWithRefs(subject, schema, refs)` | Register schema with references |
| `ValidateSchema(subject, schema)` | Check compatibility |
| `GetAllSubjects()` | List all subjects |

## Serializer Methods

| Method | Description |
|--------|-------------|
| `Serialize(subject, message)` | Serialize with cached schema ID |
| `SerializeWithSchema(subject, schema, message)` | Register/cache schema, then serialize |
| `SerializeWithSchemaRefs(...)` | Serialize with schema references |

## Deserializer Methods

| Method | Description |
|--------|-------------|
| `Deserialize(data)` | Parse wire format, return payload + schema ID |
| `DeserializeWithIndexes(data)` | Parse wire format with message indexes |

## Errors

| Error | Description |
|-------|-------------|
| `ErrSubjectRequired` | Subject is required |
| `ErrNilMessage` | Protobuf message is nil |
| `ErrSchemaRequired` | Schema text required for first serialize |
| `ErrSchemaNotCached` | Schema ID not cached; call SerializeWithSchema |
| `ErrDataTooShort` | Wire format payload too short |
| `ErrInvalidMagicByte` | Invalid magic byte (not 0x00) |
| `ErrInvalidMessageIndexes` | Invalid protobuf message indexes |

## Business Example

### Payment Event Producer

```go
const paymentSchemaProto = `syntax = "proto3";
package payment.v1;

message PaymentCompleted {
    string payment_id = 1;
    int64 amount = 2;
    string currency = 3;
    google.protobuf.Timestamp completed_at = 4;
}
`

type PaymentPublisher struct {
    producer    *franzgo.Producer
    serializer  *schemaregistry.ProtoSerializer
}

func NewPaymentPublisher(client *franzgo.Client, srClient *schemaregistry.Client) *PaymentPublisher {
    return &PaymentPublisher{
        producer:   franzgo.NewProducer(client, "payment-events"),
        serializer: schemaregistry.NewProtoSerializer(srClient),
    }
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
```

### Payment Event Consumer

```go
type PaymentConsumer struct {
    consumer      *franzgo.Consumer
    deserializer  *schemaregistry.ProtoDeserializer
}

func (c *PaymentConsumer) Start(ctx context.Context) error {
    return c.consumer.Consume(ctx, []string{"payment-events"}, func(msg *franzgo.Message) {
        payload, schemaID, err := c.deserializer.Deserialize(msg.Value)
        if err != nil {
            log.Errorw("failed to deserialize", "error", err)
            return
        }
        
        var event paymentv1.PaymentCompleted
        if err := proto.Unmarshal(payload, &event); err != nil {
            log.Errorw("failed to unmarshal protobuf", "error", err)
            return
        }
        
        c.processPayment(ctx, &event)
    })
}
```

## Schema References

For nested protobuf messages:

```go
refs := []schemaregistry.SchemaReference{
    {Name: "common.proto", Subject: "common-value", Version: 1},
}

payload, _, err := serializer.SerializeWithSchemaRefs(
    "payment-events-value",
    paymentSchemaProto,
    refs,
    event,
)
```

## Caching

The serializer caches schema IDs per subject to avoid repeated registry lookups:

```go
payload, id1, _ := serializer.SerializeWithSchema("topic-value", schema, msg1)
payload, id2, _ := serializer.Serialize("topic-value", msg2)
```

When a `.proto` file contains multiple message types, the serializer encodes the correct message-index path for each concrete `proto.Message` value.

## Testing

```bash
go test ./...
```

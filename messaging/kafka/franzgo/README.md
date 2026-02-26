# franzgo

Kafka client wrapper based on [franz-go](https://github.com/twmb/franz-go).

## Features

- Simple client, producer, and consumer abstractions
- Consumer-group auto-commit controls
- Synchronous produce helpers

## Installation

```go
go get github.com/vortex-fintech/go-lib/messaging/kafka/franzgo
```

## Quick Start

### Client

```go
import "github.com/vortex-fintech/go-lib/messaging/kafka/franzgo"

func main() {
    client, err := franzgo.NewClient(franzgo.Config{
        SeedBrokers:   []string{"localhost:9092"},
        ClientID:      "my-service",
        ConsumerGroup: "my-consumers",
    })
    if err != nil {
        panic(err)
    }
    defer client.Close()

    ctx := context.Background()
    if err := client.Ping(ctx); err != nil {
        panic(err)
    }
}
```

### Producer

```go
producer := franzgo.NewProducer(client, "payment-events")

err := producer.Produce(ctx, []byte("payment-123"), []byte(`{"status":"completed"}`))
if err != nil {
    panic(err)
}

// With headers
err = producer.ProduceWithHeaders(ctx, []byte("payment-123"), []byte(payload), []kgo.RecordHeader{
    {Key: "trace-id", Value: []byte("abc-123")},
})
```

### Consumer

```go
consumer := franzgo.NewConsumer(client, "payment-consumers")

err := consumer.Consume(ctx, []string{"payment-events"}, func(msg *franzgo.Message) {
    fmt.Printf("Received: topic=%s, partition=%d, offset=%d\n",
        msg.Topic, msg.Partition, msg.Offset)
    fmt.Printf("Key: %s, Value: %s\n", msg.Key, msg.Value)
})
```

## Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `SeedBrokers` | `[]string` | `["localhost:9092"]` | Kafka broker addresses |
| `ClientID` | `string` | `"franzgo-client"` | Client identifier |
| `ConsumerGroup` | `string` | `""` | Consumer group ID |
| `DisableAutoCommit` | `bool` | `false` | Disable group auto-commit |
| `AutoCommitMarks` | `bool` | `false` | Commit only marked records |
| `AutoCommitInterval` | `time.Duration` | `5s` | Auto-commit interval |

`DisableAutoCommit`, `AutoCommitMarks`, and `AutoCommitInterval` are valid only when `ConsumerGroup` is set.

## Message Fields

| Field | Type | Description |
|-------|------|-------------|
| `Topic` | `string` | Topic name |
| `Partition` | `int32` | Partition ID |
| `Offset` | `int64` | Message offset |
| `Key` | `[]byte` | Message key |
| `Value` | `[]byte` | Message value |
| `Headers` | `[]kgo.RecordHeader` | Message headers |
| `Timestamp` | `time.Time` | Message timestamp |

## Business Example

### Payment Event Publisher

```go
type PaymentService struct {
    producer *franzgo.Producer
}

func (s *PaymentService) PublishPaymentCompleted(ctx context.Context, paymentID string) error {
    event := PaymentCompletedEvent{
        PaymentID:   paymentID,
        CompletedAt: time.Now().UTC(),
    }
    
    payload, err := json.Marshal(event)
    if err != nil {
        return err
    }
    
    return s.producer.Produce(ctx, []byte(paymentID), payload)
}
```

### Payment Event Consumer

```go
type PaymentProcessor struct {
    consumer *franzgo.Consumer
}

func (p *PaymentProcessor) Start(ctx context.Context) error {
    return p.consumer.Consume(ctx, []string{"payment-events"}, func(msg *franzgo.Message) {
        var event PaymentCompletedEvent
        if err := json.Unmarshal(msg.Value, &event); err != nil {
            log.Errorw("failed to unmarshal event", "error", err)
            return
        }
        
        p.processEvent(ctx, event)
    })
}
```

## Testing

```bash
go test ./...

# Integration tests (requires Kafka)
KAFKA_BROKER=localhost:9092 go test -tags integration ./...
```

## Notes

- Uses franz-go's `ProduceSync` for synchronous production
- Auto-commit interval defaults to 5 seconds in consumer-group mode
- `Consume` returns fetch errors instead of silently skipping them
- Topic auto-creation is enabled in this wrapper

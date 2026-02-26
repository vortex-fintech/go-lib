package franzgo

import (
	"context"
	"errors"
	"testing"
	"time"

	kgo "github.com/twmb/franz-go/pkg/kgo"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.DisableAutoCommit {
		t.Fatal("DisableAutoCommit should be false by default")
	}
	if cfg.AutoCommitMarks {
		t.Fatal("AutoCommitMarks should be false by default")
	}
	if cfg.AutoCommitInterval != 5*time.Second {
		t.Fatalf("expected AutoCommitInterval 5s, got %s", cfg.AutoCommitInterval)
	}
}

func TestNewClient_Defaults(t *testing.T) {
	client, err := NewClient(Config{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer client.Close()

	if client.Client == nil {
		t.Fatal("client should not be nil")
	}
}

func TestNewClient_WithGroupConfig(t *testing.T) {
	cfg := Config{
		SeedBrokers:        []string{"broker1:9092", "broker2:9092"},
		ClientID:           "test-client",
		ConsumerGroup:      "test-group",
		AutoCommitInterval: 2 * time.Second,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer client.Close()

	if client.Client == nil {
		t.Fatal("client should not be nil")
	}
}

func TestNewClient_GroupOnlyOptionsWithoutGroup(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{
			name: "disable auto commit",
			cfg: Config{
				DisableAutoCommit: true,
			},
		},
		{
			name: "auto commit marks",
			cfg: Config{
				AutoCommitMarks: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(tt.cfg)
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNewClient_DisableAutoCommitWithMarks(t *testing.T) {
	_, err := NewClient(Config{
		ConsumerGroup:     "test-group",
		DisableAutoCommit: true,
		AutoCommitMarks:   true,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestNewProducer(t *testing.T) {
	client, err := NewClient(Config{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer client.Close()

	producer := NewProducer(client, "test-topic")
	if producer == nil {
		t.Fatal("producer should not be nil")
	}
	if producer.Topic() != "test-topic" {
		t.Fatalf("expected topic 'test-topic', got %q", producer.Topic())
	}
}

func TestNewProducer_DefaultTopic(t *testing.T) {
	client, err := NewClient(Config{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer client.Close()

	producer := NewProducer(client, "")
	if producer.Topic() != "default-topic" {
		t.Fatalf("expected default topic 'default-topic', got %q", producer.Topic())
	}
}

func TestProducer_ProduceBatch_Empty(t *testing.T) {
	client, err := NewClient(Config{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer client.Close()

	producer := NewProducer(client, "test-topic")
	if err := producer.ProduceBatch(context.Background(), nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestProducer_ProduceBatch_NilRecord(t *testing.T) {
	client, err := NewClient(Config{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer client.Close()

	producer := NewProducer(client, "test-topic")
	err = producer.ProduceBatch(context.Background(), []*kgo.Record{nil})
	if !errors.Is(err, ErrProducerRecordNil) {
		t.Fatalf("expected ErrProducerRecordNil, got %v", err)
	}
}

func TestProducer_ProduceBatch_DoesNotMutateInputRecords(t *testing.T) {
	client, err := NewClient(Config{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer client.Close()

	producer := NewProducer(client, "test-topic")
	rec := &kgo.Record{Key: []byte("k"), Value: []byte("v")}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_ = producer.ProduceBatch(ctx, []*kgo.Record{rec})

	if rec.Topic != "" {
		t.Fatalf("expected input record topic to remain empty, got %q", rec.Topic)
	}
}

func TestNewConsumer(t *testing.T) {
	client, err := NewClient(Config{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer client.Close()

	consumer := NewConsumer(client, "test-group")
	if consumer == nil {
		t.Fatal("consumer should not be nil")
	}
	if consumer.Group() != "test-group" {
		t.Fatalf("expected group 'test-group', got %q", consumer.Group())
	}
}

func TestConsumer_Consume_EmptyTopics(t *testing.T) {
	client, err := NewClient(Config{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer client.Close()

	consumer := NewConsumer(client, "test-group")
	err = consumer.Consume(context.Background(), []string{}, func(_ *Message) {})
	if err != nil {
		t.Fatalf("expected nil for empty topics, got: %v", err)
	}
}

func TestConsumer_Consume_NilHandler(t *testing.T) {
	client, err := NewClient(Config{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer client.Close()

	consumer := NewConsumer(client, "test-group")
	err = consumer.Consume(context.Background(), []string{"topic"}, nil)
	if !errors.Is(err, ErrConsumerHandlerNil) {
		t.Fatalf("expected ErrConsumerHandlerNil, got %v", err)
	}
}

func TestMessage_Fields(t *testing.T) {
	now := time.Now().UTC()
	msg := &Message{
		Topic:     "test-topic",
		Partition: 1,
		Offset:    100,
		Key:       []byte("test-key"),
		Value:     []byte("test-value"),
		Headers: []kgo.RecordHeader{
			{Key: "header-key", Value: []byte("header-value")},
		},
		Timestamp: now,
	}

	if msg.Topic != "test-topic" {
		t.Fatalf("expected topic 'test-topic', got %q", msg.Topic)
	}
	if msg.Partition != 1 {
		t.Fatalf("expected partition 1, got %d", msg.Partition)
	}
	if msg.Offset != 100 {
		t.Fatalf("expected offset 100, got %d", msg.Offset)
	}
	if string(msg.Key) != "test-key" {
		t.Fatalf("expected key 'test-key', got %q", msg.Key)
	}
	if string(msg.Value) != "test-value" {
		t.Fatalf("expected value 'test-value', got %q", msg.Value)
	}
	if len(msg.Headers) != 1 {
		t.Fatalf("expected 1 header, got %d", len(msg.Headers))
	}
	if !msg.Timestamp.Equal(now) {
		t.Fatalf("expected timestamp %v, got %v", now, msg.Timestamp)
	}
}

func TestClient_Close(t *testing.T) {
	client, err := NewClient(Config{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	client.Close()
	client.Close()
}

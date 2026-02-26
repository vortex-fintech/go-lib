//go:build integration

package franzgo

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func integrationBroker() string {
	if broker := os.Getenv("KAFKA_BROKER"); broker != "" {
		return broker
	}
	return "localhost:9092"
}

func TestIntegration_NewClient(t *testing.T) {
	cfg := Config{
		SeedBrokers: []string{integrationBroker()},
		ClientID:    "integration-test",
	}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		t.Fatalf("failed to ping Kafka: %v", err)
	}
}

func TestIntegration_ProduceAndConsume(t *testing.T) {
	cfg := Config{
		SeedBrokers:   []string{integrationBroker()},
		ClientID:      "integration-test",
		ConsumerGroup: fmt.Sprintf("integration-test-group-%d", time.Now().UnixNano()),
	}
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	topic := "integration-test-topic"
	producer := NewProducer(client, topic)

	key := []byte("test-key")
	value := []byte("test-value")

	if err := producer.Produce(ctx, key, value); err != nil {
		t.Fatalf("failed to produce message: %v", err)
	}

	consumer := NewConsumer(client, cfg.ConsumerGroup)

	received := make(chan *Message, 1)
	errCh := make(chan error, 1)
	go func() {
		errCh <- consumer.Consume(ctx, []string{topic}, func(msg *Message) {
			select {
			case received <- msg:
			default:
			}
		})
	}()

	select {
	case msg := <-received:
		if string(msg.Key) != "test-key" {
			t.Errorf("expected key 'test-key', got %q", msg.Key)
		}
		if string(msg.Value) != "test-value" {
			t.Errorf("expected value 'test-value', got %q", msg.Value)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for message")
	case err := <-errCh:
		if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
			t.Fatalf("consumer returned error: %v", err)
		}
	}
}

package franzgo

import (
	"context"
	"errors"
	"fmt"
	"time"

	kgo "github.com/twmb/franz-go/pkg/kgo"
)

var (
	ErrConsumerClientNil  = errors.New("consumer client is nil")
	ErrConsumerHandlerNil = errors.New("consumer handler is nil")
)

type Message struct {
	Topic     string
	Partition int32
	Offset    int64
	Key       []byte
	Value     []byte
	Headers   []kgo.RecordHeader
	Timestamp time.Time
}

type HandlerFunc func(msg *Message)

type Consumer struct {
	client *Client
	group  string
}

func NewConsumer(client *Client, group string) *Consumer {
	return &Consumer{
		client: client,
		group:  group,
	}
}

func (c *Consumer) Consume(ctx context.Context, topics []string, handler HandlerFunc) error {
	if c == nil || c.client == nil || c.client.Client == nil {
		return ErrConsumerClientNil
	}
	if handler == nil {
		return ErrConsumerHandlerNil
	}
	if len(topics) == 0 {
		return nil
	}

	c.client.AddConsumeTopics(topics...)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			fetches := c.client.PollFetches(ctx)
			if err := ctx.Err(); err != nil {
				return err
			}
			if fetches.IsClientClosed() {
				return nil
			}
			if errs := fetches.Errors(); len(errs) > 0 {
				first := errs[0]
				return fmt.Errorf("kafka fetch failed for %s[%d]: %w", first.Topic, first.Partition, first.Err)
			}

			iter := fetches.RecordIter()
			for !iter.Done() {
				record := iter.Next()
				handler(&Message{
					Topic:     record.Topic,
					Partition: record.Partition,
					Offset:    record.Offset,
					Key:       record.Key,
					Value:     record.Value,
					Headers:   record.Headers,
					Timestamp: record.Timestamp,
				})
			}
		}
	}
}

func (c *Consumer) Group() string {
	return c.group
}

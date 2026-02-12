package franzgo

import (
	"context"
	"time"

	kgo "github.com/twmb/franz-go/pkg/kgo"
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
			if fetches.IsClientClosed() {
				return nil
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

package franzgo

import (
	"context"

	kgo "github.com/twmb/franz-go/pkg/kgo"
)

type Producer struct {
	client *Client
	topic  string
}

func NewProducer(client *Client, topic string) *Producer {
	if topic == "" {
		topic = "default-topic"
	}
	return &Producer{
		client: client,
		topic:  topic,
	}
}

func (p *Producer) Produce(ctx context.Context, key, value []byte) error {
	record := kgo.Record{
		Topic: p.topic,
		Key:   key,
		Value: value,
	}
	return p.client.ProduceSync(ctx, &record).FirstErr()
}

func (p *Producer) ProduceWithHeaders(ctx context.Context, key, value []byte, headers []kgo.RecordHeader) error {
	record := kgo.Record{
		Topic:   p.topic,
		Key:     key,
		Value:   value,
		Headers: headers,
	}
	return p.client.ProduceSync(ctx, &record).FirstErr()
}

func (p *Producer) ProduceBatch(ctx context.Context, records []*kgo.Record) error {
	return p.client.ProduceSync(ctx, records...).FirstErr()
}

func (p *Producer) Topic() string {
	return p.topic
}

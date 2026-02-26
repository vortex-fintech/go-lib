package franzgo

import (
	"context"
	"errors"

	kgo "github.com/twmb/franz-go/pkg/kgo"
)

var (
	ErrProducerClientNil = errors.New("producer client is nil")
	ErrProducerRecordNil = errors.New("producer record is nil")
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
	if p == nil || p.client == nil || p.client.Client == nil {
		return ErrProducerClientNil
	}

	record := kgo.Record{
		Topic: p.topic,
		Key:   key,
		Value: value,
	}
	return p.client.ProduceSync(ctx, &record).FirstErr()
}

func (p *Producer) ProduceWithHeaders(ctx context.Context, key, value []byte, headers []kgo.RecordHeader) error {
	if p == nil || p.client == nil || p.client.Client == nil {
		return ErrProducerClientNil
	}

	record := kgo.Record{
		Topic:   p.topic,
		Key:     key,
		Value:   value,
		Headers: headers,
	}
	return p.client.ProduceSync(ctx, &record).FirstErr()
}

func (p *Producer) ProduceBatch(ctx context.Context, records []*kgo.Record) error {
	if p == nil || p.client == nil || p.client.Client == nil {
		return ErrProducerClientNil
	}
	if len(records) == 0 {
		return nil
	}

	batch := make([]*kgo.Record, 0, len(records))
	for _, record := range records {
		if record == nil {
			return ErrProducerRecordNil
		}

		copyRecord := *record
		if copyRecord.Topic == "" {
			copyRecord.Topic = p.topic
		}
		batch = append(batch, &copyRecord)
	}

	return p.client.ProduceSync(ctx, batch...).FirstErr()
}

func (p *Producer) Topic() string {
	return p.topic
}

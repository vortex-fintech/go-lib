package franzgo

import (
	"context"
	"time"

	kgo "github.com/twmb/franz-go/pkg/kgo"
)

type Client struct {
	*kgo.Client
}

type Config struct {
	SeedBrokers     []string
	ClientID        string
	ConsumerGroup   string
	AutoCommit      bool
	AutoCommitMarks bool
}

func DefaultConfig() Config {
	return Config{
		AutoCommit:      true,
		AutoCommitMarks: true,
	}
}

func NewClient(cfg Config) (*Client, error) {
	if len(cfg.SeedBrokers) == 0 {
		cfg.SeedBrokers = []string{"localhost:9092"}
	}
	if cfg.ClientID == "" {
		cfg.ClientID = "franzgo-client"
	}

	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.SeedBrokers...),
		kgo.ClientID(cfg.ClientID),
		kgo.AllowAutoTopicCreation(),
	}

	if cfg.ConsumerGroup != "" {
		opts = append(opts, kgo.ConsumerGroup(cfg.ConsumerGroup))
	}

	if cfg.AutoCommit {
		opts = append(opts, kgo.AutoCommitInterval(5*time.Second))
	}
	if cfg.AutoCommitMarks {
		opts = append(opts, kgo.AutoCommitMarks())
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, err
	}

	return &Client{Client: client}, nil
}

func (c *Client) Close() {
	c.Client.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	return nil
}

package franzgo

import (
	"context"
	"errors"
	"time"

	kgo "github.com/twmb/franz-go/pkg/kgo"
)

type Client struct {
	*kgo.Client
}

type Config struct {
	SeedBrokers        []string
	ClientID           string
	ConsumerGroup      string
	DisableAutoCommit  bool
	AutoCommitMarks    bool
	AutoCommitInterval time.Duration
}

func DefaultConfig() Config {
	return Config{
		AutoCommitInterval: 5 * time.Second,
	}
}

func NewClient(cfg Config) (*Client, error) {
	if cfg.ConsumerGroup == "" {
		if cfg.DisableAutoCommit {
			return nil, errors.New("disable auto commit requires consumer group")
		}
		if cfg.AutoCommitMarks {
			return nil, errors.New("auto commit marks requires consumer group")
		}
	}

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

		if cfg.DisableAutoCommit {
			if cfg.AutoCommitMarks {
				return nil, errors.New("auto commit marks cannot be enabled when auto commit is disabled")
			}
			opts = append(opts, kgo.DisableAutoCommit())
		} else {
			interval := cfg.AutoCommitInterval
			if interval <= 0 {
				interval = 5 * time.Second
			}
			opts = append(opts, kgo.AutoCommitInterval(interval))
			if cfg.AutoCommitMarks {
				opts = append(opts, kgo.AutoCommitMarks())
			}
		}
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, err
	}

	return &Client{Client: client}, nil
}

func (c *Client) Close() {
	if c == nil || c.Client == nil {
		return
	}
	c.Client.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.Client == nil {
		return errors.New("client is nil")
	}
	return c.Client.Ping(ctx)
}

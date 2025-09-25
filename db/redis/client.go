package redis

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/redis/go-redis/v9"
)

var NewUniversal = func(opt *redis.UniversalOptions) redis.UniversalClient {
	return redis.NewUniversalClient(opt)
}

func NewRedisClient(ctx context.Context, cfg Config) (redis.UniversalClient, error) {
	addrs := cfg.Addrs
	if len(addrs) == 0 && cfg.Addr != "" {
		addrs = []string{cfg.Addr}
	}

	opt := &redis.UniversalOptions{
		Addrs:        addrs,
		MasterName:   cfg.MasterName,
		DB:           cfg.DB,
		Username:     cfg.Username,
		Password:     cfg.Password,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	if cfg.TLSEnabled {
		opt.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	rdb := NewUniversal(opt)

	pingTimeout := cfg.DialTimeout
	if pingTimeout <= 0 {
		pingTimeout = 3 * time.Second
	}
	c, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	if err := rdb.Ping(c).Err(); err != nil {
		_ = rdb.Close()
		return nil, err
	}
	return rdb, nil
}

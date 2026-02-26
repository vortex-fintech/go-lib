package redis

import (
	"context"
	"crypto/tls"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var NewUniversal = func(opt *redis.UniversalOptions) redis.UniversalClient {
	return redis.NewUniversalClient(opt)
}

func NewRedisClient(ctx context.Context, cfg Config) (redis.UniversalClient, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	mode := normalizeMode(cfg.Mode)
	addrs := normalizeAddrs(cfg)
	if err := validateConfig(cfg, mode, addrs); err != nil {
		return nil, err
	}

	opt := &redis.UniversalOptions{
		Addrs:        addrs,
		MasterName:   strings.TrimSpace(cfg.MasterName),
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

package postgres

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// test hooks (подменяемы в unit-тестах)
var (
	newPool  = pgxpool.NewWithConfig
	pingPool = func(ctx context.Context, p *pgxpool.Pool) error { return p.Ping(ctx) }
)

type Client struct {
	Pool *pgxpool.Pool
}

func Open(ctx context.Context, cfg Config) (*Client, error) {
	pcfg, err := pgxpool.ParseConfig(buildURL(cfg))
	if err != nil {
		return nil, err
	}

	if cfg.MaxConns > 0 {
		pcfg.MaxConns = cfg.MaxConns
	}
	pcfg.MinConns = cfg.MinConns
	if cfg.MaxConnLifetime > 0 {
		pcfg.MaxConnLifetime = cfg.MaxConnLifetime
	}
	if cfg.MaxConnIdleTime > 0 {
		pcfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	}
	if cfg.HealthCheckPeriod > 0 {
		pcfg.HealthCheckPeriod = cfg.HealthCheckPeriod
	}

	pool, err := newPool(ctx, pcfg)
	if err != nil {
		return nil, err
	}

	// быстрый health-check
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pingPool(pingCtx, pool); err != nil {
		if pool != nil {
			pool.Close()
		}
		return nil, err
	}

	return &Client{Pool: pool}, nil
}

func (c *Client) Close() {
	if c != nil && c.Pool != nil {
		c.Pool.Close()
	}
}

func buildURL(cfg Config) string {
	if strings.TrimSpace(cfg.URL) == "" {
		return ""
	}
	if len(cfg.Params) == 0 {
		return cfg.URL
	}
	u, err := url.Parse(cfg.URL)
	if err != nil {
		return cfg.URL
	}
	q := u.Query()
	for k, v := range cfg.Params {
		if v != "" {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

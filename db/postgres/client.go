package postgres

import (
	"context"
	"fmt"
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

// Открыть по высокоуровневому Config (URL + pool options).
func Open(ctx context.Context, cfg Config) (*Client, error) {
	dsn := buildURL(cfg)
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("postgres: empty URL")
	}

	pcfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	// Параметры пула
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

// Открыть по структурному DBConfig (host/port/user/...).
// Собирает URL внутри и переносит пул-настройки из DBConfig.
func OpenWithDBConfig(ctx context.Context, dbCfg DBConfig) (*Client, error) {
	dsn := buildURLFromDB(dbCfg)

	return Open(ctx, Config{
		URL: dsn,
		// Маппим настройки пула
		MaxConns:          int32(dbCfg.MaxOpenConns),
		MinConns:          int32(dbCfg.MaxIdleConns),
		MaxConnLifetime:   dbCfg.ConnMaxLifetime,
		MaxConnIdleTime:   dbCfg.ConnMaxIdleTime,
		HealthCheckPeriod: 0, // оставим по умолчанию
	})
}

func (c *Client) Close() {
	if c != nil && c.Pool != nil {
		c.Pool.Close()
	}
}

// --- helpers ---

// buildURL — применяет cfg.Params к cfg.URL (если заданы).
func buildURL(cfg Config) string {
	base := strings.TrimSpace(cfg.URL)
	if base == "" {
		return ""
	}
	if len(cfg.Params) == 0 {
		return base
	}
	u, err := url.Parse(base)
	if err != nil {
		return base
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

// buildURLFromDB — собирает postgres DSN из структурного DBConfig.
func buildURLFromDB(c DBConfig) string {
	u := &url.URL{
		Scheme: "postgres",
		Host:   fmt.Sprintf("%s:%s", c.Host, c.Port),
		Path:   "/" + strings.TrimPrefix(c.DBName, "/"),
	}
	if c.User != "" || c.Password != "" {
		u.User = url.UserPassword(c.User, c.Password)
	}
	q := u.Query()
	if c.SSLMode != "" {
		q.Set("sslmode", c.SSLMode)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

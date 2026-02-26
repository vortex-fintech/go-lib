package postgres

import (
	"context"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Test hooks (replaceable in unit tests).
var (
	newPool  = pgxpool.NewWithConfig
	pingPool = func(ctx context.Context, p *pgxpool.Pool) error { return p.Ping(ctx) }
)

type Client struct {
	Pool *pgxpool.Pool
}

// Open creates a client from high-level Config (URL + pool options).
func Open(ctx context.Context, cfg Config) (*Client, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	dsn := buildURL(cfg)

	pcfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	// Pool options
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

	// Useful defaults: pg_stat_activity visibility and unified timezone.
	if pcfg.ConnConfig != nil {
		// v5: runtime params are stored in ConnConfig.Config.RuntimeParams.
		if pcfg.ConnConfig.Config.RuntimeParams == nil {
			pcfg.ConnConfig.Config.RuntimeParams = map[string]string{}
		}
		if _, ok := pcfg.ConnConfig.Config.RuntimeParams["application_name"]; !ok {
			pcfg.ConnConfig.Config.RuntimeParams["application_name"] = "go-lib-pgxpool"
		}
		if _, ok := pcfg.ConnConfig.Config.RuntimeParams["TimeZone"]; !ok {
			pcfg.ConnConfig.Config.RuntimeParams["TimeZone"] = "UTC"
		}
	}

	pool, err := newPool(ctx, pcfg)
	if err != nil {
		return nil, err
	}

	// Fast health check
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

// OpenWithDBConfig creates a client from structured DBConfig (host/port/user/...)
// and maps pool options to Config.
func OpenWithDBConfig(ctx context.Context, dbCfg DBConfig) (*Client, error) {
	if err := dbCfg.validate(); err != nil {
		return nil, err
	}

	dsn := buildURLFromDB(dbCfg)
	return Open(ctx, Config{
		URL: dsn,
		// Map pool options.
		// DBConfig.MaxIdleConns is used as a minimum warm pool size (MinConns).
		MaxConns:          int32(dbCfg.MaxOpenConns),
		MinConns:          int32(dbCfg.MaxIdleConns),
		MaxConnLifetime:   dbCfg.ConnMaxLifetime,
		MaxConnIdleTime:   dbCfg.ConnMaxIdleTime,
		HealthCheckPeriod: 0, // keep default
	})
}

func (c *Client) Close() {
	if c != nil && c.Pool != nil {
		c.Pool.Close()
	}
}

// --- helpers ---

// buildURL applies cfg.Params to cfg.URL when params are provided.
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

// buildURLFromDB builds postgres DSN from structured DBConfig.
// It is IPv6-safe thanks to net.JoinHostPort.
func buildURLFromDB(c DBConfig) string {
	u := &url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(c.Host, c.Port),
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

package postgres

import (
	"errors"
	"strings"
	"time"
)

// DBConfig is a low-level connection config (host/port/...).
// It is convenient for apps: load from env and pass to the library.
type DBConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int           // max connections in pool
	MaxIdleConns    int           // minimum warm connections in pool
	ConnMaxLifetime time.Duration // max connection lifetime
	ConnMaxIdleTime time.Duration // max idle lifetime
}

// Config is a high-level connection config based on URL.
// Useful for complex DSN cases and backward compatibility.
type Config struct {
	URL    string            // postgres://user:pass@host:port/dbname?sslmode=disable
	Params map[string]string // extra URL params (override query)

	// Pool options
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

var (
	errEmptyURL                = errors.New("postgres: empty URL")
	errHostRequired            = errors.New("postgres: host is required")
	errPortRequired            = errors.New("postgres: port is required")
	errDBNameRequired          = errors.New("postgres: db name is required")
	errNegativeMaxConns        = errors.New("postgres: max conns must be >= 0")
	errNegativeMinConns        = errors.New("postgres: min conns must be >= 0")
	errMinConnsExceedsMaxConns = errors.New("postgres: min conns must be <= max conns")
)

func (c Config) validate() error {
	if strings.TrimSpace(c.URL) == "" {
		return errEmptyURL
	}
	if c.MaxConns < 0 {
		return errNegativeMaxConns
	}
	if c.MinConns < 0 {
		return errNegativeMinConns
	}
	if c.MaxConns > 0 && c.MinConns > c.MaxConns {
		return errMinConnsExceedsMaxConns
	}
	return nil
}

func (c DBConfig) validate() error {
	if strings.TrimSpace(c.Host) == "" {
		return errHostRequired
	}
	if strings.TrimSpace(c.Port) == "" {
		return errPortRequired
	}
	if strings.TrimSpace(c.DBName) == "" {
		return errDBNameRequired
	}
	if c.MaxOpenConns < 0 {
		return errNegativeMaxConns
	}
	if c.MaxIdleConns < 0 {
		return errNegativeMinConns
	}
	if c.MaxOpenConns > 0 && c.MaxIdleConns > c.MaxOpenConns {
		return errMinConnsExceedsMaxConns
	}
	return nil
}

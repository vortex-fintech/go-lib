package postgres

import "time"

type Config struct {
	// Предпочтительно полная URL-строка:
	// postgres://user:pass@host:port/db?sslmode=require
	URL    string
	Params map[string]string // доп. query-параметры, если нужно

	// Пул соединений
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

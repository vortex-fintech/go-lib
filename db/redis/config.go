package redis

import "time"

type Mode string

const (
	ModeSingle   Mode = "single"
	ModeSentinel Mode = "sentinel"
	ModeCluster  Mode = "cluster"
)

type Config struct {
	// Какой режим используем
	Mode Mode

	// Адреса:
	// - single: можно указать Addr ИЛИ Addrs[0]
	// - sentinel: список адресов sentinel-нод в Addrs
	// - cluster: список адресов cluster-нод в Addrs
	Addr  string
	Addrs []string

	// Только для sentinel
	MasterName string

	// Для single/sentinel
	DB int

	// Аутентификация
	Username string
	Password string

	// Таймауты и пул
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
	MinIdleConns int

	// TLS
	TLSEnabled bool
	// При необходимости можно добавить:
	// TLSInsecureSkipVerify bool
}

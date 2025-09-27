package redis

import "time"

type Mode = string

const (
	ModeSingle   Mode = "single"
	ModeSentinel Mode = "sentinel"
	ModeCluster  Mode = "cluster"
)

type Config struct {
	Mode         string
	Addr         string
	Addrs        []string
	MasterName   string
	DB           int
	Username     string
	Password     string
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
	MinIdleConns int
	TLSEnabled   bool
}

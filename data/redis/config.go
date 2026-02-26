package redis

import (
	"errors"
	"strings"
	"time"
)

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

var (
	errAddressRequired      = errors.New("redis: address is required")
	errUnsupportedMode      = errors.New("redis: unsupported mode")
	errMasterNameRequired   = errors.New("redis: master name is required for sentinel mode")
	errMasterNameUnexpected = errors.New("redis: master name is only valid for sentinel mode")
	errSingleModeAddrCount  = errors.New("redis: single mode requires exactly one address")
	errClusterModeAddrCount = errors.New("redis: cluster mode requires at least two addresses")
	errClusterDBUnsupported = errors.New("redis: db must be 0 in cluster mode")
	errInvalidDB            = errors.New("redis: db must be >= 0")
)

func normalizeMode(v string) Mode {
	mode := strings.ToLower(strings.TrimSpace(v))
	if mode == "" {
		return ModeSingle
	}
	return Mode(mode)
}

func normalizeAddrs(cfg Config) []string {
	out := make([]string, 0, len(cfg.Addrs)+1)
	for _, a := range cfg.Addrs {
		a = strings.TrimSpace(a)
		if a != "" {
			out = append(out, a)
		}
	}
	if len(out) == 0 {
		if a := strings.TrimSpace(cfg.Addr); a != "" {
			out = append(out, a)
		}
	}
	return out
}

func validateConfig(cfg Config, mode Mode, addrs []string) error {
	if cfg.DB < 0 {
		return errInvalidDB
	}
	if len(addrs) == 0 {
		return errAddressRequired
	}

	switch mode {
	case ModeSingle:
		if len(addrs) != 1 {
			return errSingleModeAddrCount
		}
		if strings.TrimSpace(cfg.MasterName) != "" {
			return errMasterNameUnexpected
		}
		return nil
	case ModeCluster:
		if len(addrs) < 2 {
			return errClusterModeAddrCount
		}
		if strings.TrimSpace(cfg.MasterName) != "" {
			return errMasterNameUnexpected
		}
		if cfg.DB != 0 {
			return errClusterDBUnsupported
		}
		return nil
	case ModeSentinel:
		if strings.TrimSpace(cfg.MasterName) == "" {
			return errMasterNameRequired
		}
		return nil
	default:
		return errUnsupportedMode
	}
}

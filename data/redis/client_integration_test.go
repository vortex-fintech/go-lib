//go:build integration

package redis_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	redispkg "github.com/vortex-fintech/go-lib/data/redis"
)

func TestNewRedisClient_Single_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := redispkg.Config{
		Mode:         redispkg.ModeSingle,
		Addr:         integrationRedisAddr(),
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	}

	c, err := redispkg.NewRedisClient(ctx, cfg)
	require.NoError(t, err)
	defer func() {
		_ = c.Close()
	}()

	key := fmt.Sprintf("go-lib:data:redis:it:%d", time.Now().UnixNano())
	require.NoError(t, c.Set(ctx, key, "ok", 30*time.Second).Err())

	v, err := c.Get(ctx, key).Result()
	require.NoError(t, err)
	require.Equal(t, "ok", v)
}

func TestNewRedisClient_Sentinel_Integration(t *testing.T) {
	addrs, ok := csvEnv("REDIS_TEST_SENTINEL_ADDRS")
	master := strings.TrimSpace(os.Getenv("REDIS_TEST_SENTINEL_MASTER"))
	if !ok || master == "" {
		t.Skip("set REDIS_TEST_SENTINEL_ADDRS and REDIS_TEST_SENTINEL_MASTER to run sentinel integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c, err := redispkg.NewRedisClient(ctx, redispkg.Config{
		Mode:         redispkg.ModeSentinel,
		Addrs:        addrs,
		MasterName:   master,
		DB:           0,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
	require.NoError(t, err)
	defer func() {
		_ = c.Close()
	}()

	key := fmt.Sprintf("go-lib:data:redis:sentinel:it:%d", time.Now().UnixNano())
	require.NoError(t, c.Set(ctx, key, "ok", 30*time.Second).Err())

	v, err := c.Get(ctx, key).Result()
	require.NoError(t, err)
	require.Equal(t, "ok", v)
}

func TestNewRedisClient_Cluster_Integration(t *testing.T) {
	addrs, ok := csvEnv("REDIS_TEST_CLUSTER_ADDRS")
	if !ok {
		t.Skip("set REDIS_TEST_CLUSTER_ADDRS to run cluster integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c, err := redispkg.NewRedisClient(ctx, redispkg.Config{
		Mode:         redispkg.ModeCluster,
		Addrs:        addrs,
		DB:           0,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
	require.NoError(t, err)
	defer func() {
		_ = c.Close()
	}()

	key := fmt.Sprintf("go-lib:data:redis:cluster:it:%d", time.Now().UnixNano())
	require.NoError(t, c.Set(ctx, key, "ok", 30*time.Second).Err())

	v, err := c.Get(ctx, key).Result()
	require.NoError(t, err)
	require.Equal(t, "ok", v)
}

func integrationRedisAddr() string {
	if v := os.Getenv("REDIS_TEST_ADDR"); v != "" {
		return v
	}
	return "localhost:6380"
}

func csvEnv(key string) ([]string, bool) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil, false
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

// go-lib/security/replay/replay.go
package replay

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Checker — абстракция анти-replay стора.
// Должен вернуть true, если JTI уже встречался (т.е. это replay).
type Checker interface {
	SeenJTI(ctx context.Context, namespace, jti string, ttl time.Duration) (seen bool, err error)
}

var (
	ErrInvalidTTL     = errors.New("replay: ttl must be greater than zero")
	ErrNilRedisClient = errors.New("replay: redis client is nil")
)

// --------------------------- Redis реализация ---------------------------

type RedisOptions struct {
	// Префикс ключа в Redis, по умолчанию "obo:jti".
	Prefix string
	// FailOpen: при ошибке Redis считать, что replay НЕТ (пропускать).
	// По умолчанию false (fail-closed): ошибка = блокируем (считаем replay).
	FailOpen bool
}

type RedisChecker struct {
	rdb      redis.UniversalClient
	prefix   string
	failOpen bool
}

func NewRedisChecker(rdb redis.UniversalClient, opt RedisOptions) *RedisChecker {
	prefix := opt.Prefix
	if prefix == "" {
		prefix = "obo:jti"
	}
	return &RedisChecker{
		rdb:      rdb,
		prefix:   prefix,
		failOpen: opt.FailOpen,
	}
}

func (r *RedisChecker) SeenJTI(ctx context.Context, namespace, jti string, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		if r.failOpen {
			return false, nil
		}
		return true, ErrInvalidTTL
	}
	if r.rdb == nil {
		if r.failOpen {
			return false, nil
		}
		return true, ErrNilRedisClient
	}

	key := fmt.Sprintf("%s:%s:%s", r.prefix, namespace, jti)
	ok, err := r.rdb.SetNX(ctx, key, 1, ttl).Result()
	if err != nil {
		// Ошибка Redis: политикой решаем, пропускать (fail-open) или блокировать (fail-closed).
		if r.failOpen {
			return false, nil
		}
		return true, err
	}
	// SetNX вернул false => ключ уже был => это replay.
	return !ok, nil
}

// AsAuthzCallback — адаптер под подпись SeenJTI, которую ждёт authz.Config (func(string) bool).
// ctx используется Background, т.к. колбэк короткий и без I/O блокировок кроме Redis.
func (r *RedisChecker) AsAuthzCallback(namespace string, ttl time.Duration) func(string) bool {
	return func(jti string) bool {
		seen, _ := r.SeenJTI(context.Background(), namespace, jti, ttl)
		return seen
	}
}

// ------------------------ In-Memory (dev / fallback) ------------------------

type MemoryOptions struct {
	TTL      time.Duration
	MaxItems int
}

type InMemoryChecker struct {
	mu    sync.Mutex
	items map[string]time.Time
	opt   MemoryOptions
}

func NewInMemoryChecker(opt MemoryOptions) *InMemoryChecker {
	return &InMemoryChecker{
		items: make(map[string]time.Time),
		opt:   opt,
	}
}

func (m *InMemoryChecker) SeenJTI(_ context.Context, namespace, jti string, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ttl <= 0 {
		ttl = m.opt.TTL
	}
	if ttl <= 0 {
		return true, ErrInvalidTTL
	}
	now := time.Now()
	key := fmt.Sprintf("%s:%s:%s", "obo:jti", namespace, jti)

	m.gc(now)

	if exp, ok := m.items[key]; ok && exp.After(now) {
		return true, nil
	}
	m.items[key] = now.Add(ttl)
	return false, nil
}

func (m *InMemoryChecker) gc(now time.Time) {
	if len(m.items) == 0 {
		return
	}

	for k, exp := range m.items {
		if !exp.After(now) {
			delete(m.items, k)
		}
	}

	if m.opt.MaxItems > 0 && len(m.items) >= m.opt.MaxItems {
		oldest := ""
		oldestExp := now.Add(100 * 365 * 24 * time.Hour)
		for k, exp := range m.items {
			if exp.Before(oldestExp) {
				oldestExp = exp
				oldest = k
			}
		}
		if oldest != "" {
			delete(m.items, oldest)
		}
	}
}

func (m *InMemoryChecker) AsAuthzCallback(namespace string, ttl time.Duration) func(string) bool {
	return func(jti string) bool {
		seen, _ := m.SeenJTI(context.Background(), namespace, jti, ttl)
		return seen
	}
}

// go-lib/security/replay/replay.go
package replay

import (
	"context"
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
	TTL      time.Duration // сколько держим JTI
	MaxItems int           // мягкий лимит (для GC). 0 — без лимита.
	FailOpen bool          // семантика при внутренних ошибках (на практике не нужна)
}

type InMemoryChecker struct {
	mu       sync.Mutex
	items    map[string]time.Time // key -> expiresAt
	opt      MemoryOptions
	prefix   string
	failOpen bool
}

func NewInMemoryChecker(opt MemoryOptions) *InMemoryChecker {
	return &InMemoryChecker{
		items:    make(map[string]time.Time),
		opt:      opt,
		prefix:   "obo:jti",
		failOpen: opt.FailOpen,
	}
}

func (m *InMemoryChecker) SeenJTI(_ context.Context, namespace, jti string, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ttl <= 0 {
		ttl = m.opt.TTL
	}
	now := time.Now()
	key := fmt.Sprintf("%s:%s:%s", m.prefix, namespace, jti)

	// GC простейший: при каждом вызове чистим просроченные; при переполнении — дополнительная чистка.
	if len(m.items) > 0 {
		for k, exp := range m.items {
			if !exp.After(now) {
				delete(m.items, k)
			}
		}
	}
	if m.opt.MaxItems > 0 && len(m.items) >= m.opt.MaxItems {
		// Наивный trim: удалим часть самых старых по сроку (простое проходное удаление)
		limit := len(m.items) - m.opt.MaxItems + 1
		for k, exp := range m.items {
			if !exp.After(now) {
				delete(m.items, k)
				limit--
				if limit <= 0 {
					break
				}
			}
		}
	}

	if exp, ok := m.items[key]; ok && exp.After(now) {
		// уже есть и не истёк — replay
		return true, nil
	}
	m.items[key] = now.Add(ttl)
	return false, nil
}

func (m *InMemoryChecker) AsAuthzCallback(namespace string, ttl time.Duration) func(string) bool {
	return func(jti string) bool {
		seen, _ := m.SeenJTI(context.Background(), namespace, jti, ttl)
		return seen
	}
}

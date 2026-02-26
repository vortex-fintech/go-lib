package metadata

import (
	"context"
	"strings"

	gmd "google.golang.org/grpc/metadata"
)

const (
	HeaderAuthorization = "authorization" // "Bearer <token>"
	HeaderPoP           = "x-pop"         // x5t#S256 клиента (mTLS PoP)
	HeaderAZP           = "x-azp"         // authorized party (источник клиента)
)

// WithBearer добавляет/заменяет Authorization: Bearer <token>.
func WithBearer(ctx context.Context, token string) context.Context {
	tok := strings.TrimSpace(token)
	if tok == "" {
		return ctx
	}
	if !strings.HasPrefix(strings.ToLower(tok), "bearer ") {
		tok = "Bearer " + tok
	}
	return mergeOutgoing(ctx, map[string]string{HeaderAuthorization: tok})
}

// WithPoP добавляет X-PoP = x5t#S256 (без кавычек).
func WithPoP(ctx context.Context, x5tS256 string) context.Context {
	x := strings.TrimSpace(x5tS256)
	if x == "" {
		return ctx
	}
	return mergeOutgoing(ctx, map[string]string{HeaderPoP: x})
}

// WithAZP добавляет X-AZP (источник клиента).
func WithAZP(ctx context.Context, azp string) context.Context {
	a := strings.TrimSpace(azp)
	if a == "" {
		return ctx
	}
	return mergeOutgoing(ctx, map[string]string{HeaderAZP: a})
}

// Get читает одно значение ключа из incoming/outgoing MD (приоритет incoming).
func Get(ctx context.Context, key string) string {
	if ctx == nil {
		return ""
	}
	if md, ok := gmd.FromIncomingContext(ctx); ok {
		if v := md.Get(key); len(v) > 0 {
			return v[0]
		}
	}
	if md, ok := gmd.FromOutgoingContext(ctx); ok {
		if v := md.Get(key); len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

func GetAll(ctx context.Context, key string) []string {
	if ctx == nil {
		return nil
	}
	if md, ok := gmd.FromIncomingContext(ctx); ok {
		if v := md.Get(key); len(v) > 0 {
			return v
		}
	}
	if md, ok := gmd.FromOutgoingContext(ctx); ok {
		if v := md.Get(key); len(v) > 0 {
			return v
		}
	}
	return nil
}

// mergeOutgoing мерджит ключи в OutgoingContext (перезаписывая одноимённые).
func mergeOutgoing(ctx context.Context, kv map[string]string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	old, _ := gmd.FromOutgoingContext(ctx)
	cp := old.Copy()
	for k, v := range kv {
		low := strings.ToLower(k)
		cp.Set(low, v)
	}
	return gmd.NewOutgoingContext(ctx, cp)
}

package authz

import "context"

// тип для ключей контекста (не экспортируем, чтобы избежать коллизий)
type ctxKey string

// единый ключ для хранения всей Identity
const keyIdentity ctxKey = "authz.identity"

// Identity — то, что прокидываем в бизнес-логику
type Identity struct {
	UserID string
	Scopes []string
	SID    string
}

// WithIdentity кладёт всю Identity в context
func WithIdentity(ctx context.Context, id Identity) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, keyIdentity, id)
}

// IdentityFrom достаёт Identity из context
func IdentityFrom(ctx context.Context) (Identity, bool) {
	v := ctx.Value(keyIdentity)
	if v == nil {
		return Identity{}, false
	}
	id, ok := v.(Identity)
	return id, ok
}

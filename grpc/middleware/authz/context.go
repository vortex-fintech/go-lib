package authz

import (
	"context"

	"github.com/google/uuid"
	errs "github.com/vortex-fintech/go-lib/errors"
)

// тип для ключей контекста (не экспортируем, чтобы избежать коллизий)
type ctxKey string

// единый ключ для хранения всей Identity
const keyIdentity ctxKey = "authz.identity"

// Identity — то, что прокидываем в бизнес-логику
type Identity struct {
	UserID   uuid.UUID
	Scopes   []string
	SID      string
	DeviceID string
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

// RequireIdentity — достаёт Identity из контекста или вернёт стандартизированную ошибку.
func RequireIdentity(ctx context.Context) (Identity, error) {
	id, ok := IdentityFrom(ctx)
	if !ok || id.UserID == uuid.Nil {
		return Identity{}, errs.Internal().
			WithReason("missing_identity").
			WithDetail("auth", "user identity not found in context")
	}
	return id, nil
}

// RequireUserID — краткий вариант, если нужен только UUID.
func RequireUserID(ctx context.Context) (uuid.UUID, error) {
	id, err := RequireIdentity(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	return id.UserID, nil
}

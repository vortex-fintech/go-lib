// go-lib/authz/identity.go
package authz

import (
	"context"

	"github.com/google/uuid"
	errs "github.com/vortex-fintech/go-lib/errors"
	libjwt "github.com/vortex-fintech/go-lib/security/jwt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// тип для ключей контекста (не экспортируем, чтобы избежать коллизий)
type ctxKey string

const (
	keyIdentity ctxKey = "authz.identity"
	keyClaims   ctxKey = "authz.jwt.claims"
)

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

// WithClaims — положить полные JWT claims в контекст (если где-то нужны wallet_id/azp/acr).
func WithClaims(ctx context.Context, cl *libjwt.Claims) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, keyClaims, cl)
}

// ClaimsFrom — достать claims из контекста.
func ClaimsFrom(ctx context.Context) (*libjwt.Claims, bool) {
	v := ctx.Value(keyClaims)
	if v == nil {
		return nil, false
	}
	cl, ok := v.(*libjwt.Claims)
	return cl, ok
}

// RequireWalletID — достаёт wallet_id из положенных в контекст OBO-claims.
// Возвращает PermissionDenied, если контекст пустой или wallet_id отсутствует.
func RequireWalletID(ctx context.Context) (string, error) {
	cl, ok := ClaimsFrom(ctx)
	if !ok || cl == nil {
		return "", status.Error(codes.PermissionDenied, "missing wallet context")
	}
	if cl.WalletID == "" {
		return "", status.Error(codes.PermissionDenied, "missing wallet_id")
	}
	return cl.WalletID, nil
}

// RequireWalletMatch — сверяет wallet_id из контекста с ожидаемым значением want.
// Если want пустой — проверка считается пройденной.
func RequireWalletMatch(ctx context.Context, want string) error {
	if want == "" {
		return nil
	}
	got, err := RequireWalletID(ctx)
	if err != nil {
		return err
	}
	if got != want {
		return status.Error(codes.PermissionDenied, "wallet mismatch")
	}
	return nil
}

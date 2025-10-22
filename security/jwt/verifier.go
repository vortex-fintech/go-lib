package jwt

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Сентинел-ошибки (их удобно матчить в вызывающем коде).
var (
	ErrBadSubject          = errors.New("jwt: bad subject")
	ErrAudMismatch         = errors.New("jwt: aud mismatch")
	ErrMissingActor        = errors.New("jwt: missing actor")
	ErrActorMismatch       = errors.New("jwt: actor mismatch")
	ErrExpired             = errors.New("jwt: token expired")
	ErrIATInFuture         = errors.New("jwt: iat in the future")
	ErrTTLTooLong          = errors.New("jwt: ttl too long")
	ErrMissingJTI          = errors.New("jwt: missing jti")
	ErrReplay              = errors.New("jwt: replay detected")
	ErrMTLSBindingMismatch = errors.New("jwt: mtls binding mismatch")
	ErrMissingScopes       = errors.New("jwt: missing scopes")
)

//
// Модель OBO-токена (без nbf, с коротким TTL, scopes-только-массив).
//

// Actor (RFC 8693) — кто обменял токен (обычно клиент-сервис, напр. "api-gateway").
type Actor struct {
	Sub string `json:"sub"`
}

// PoP-binding (RFC 7800) — привязка к клиентскому сертификату.
type Cnf struct {
	X5tS256 string `json:"x5t#S256,omitempty"`
}

// Claims — внутренний OBO-токен.
type Claims struct {
	Issuer   string   `json:"iss"` // https://sso.vortex.internal
	Subject  string   `json:"sub"` // UUID пользователя
	Audience []string `json:"aud"` // Ровно один сервис: ["wallet"]

	Iat int64 `json:"iat"` // unix seconds
	Exp int64 `json:"exp"` // unix seconds

	Sid string `json:"sid,omitempty"`
	Jti string `json:"jti,omitempty"`

	// Скоупы (внутренний формат)
	Scopes []string `json:"scopes,omitempty"` // ["wallet:read","payments:create"]

	// Семантика OBO
	Azp string `json:"azp,omitempty"` // кто получил user access от SSO (напр. "frontend-web")
	Act *Actor `json:"act,omitempty"` // кто обменял токен (напр. "api-gateway")

	// Proof-of-Possession (mTLS)
	Cnf   *Cnf   `json:"cnf,omitempty"`    // mTLS PoP
	SrcTH string `json:"src_th,omitempty"` // b64url(sha256 исходного access), опционально

	// Контекст аутентификации
	ACR string   `json:"acr,omitempty"`
	AMR []string `json:"amr,omitempty"`

	DeviceID string `json:"device_id,omitempty"`
}

func (c Claims) ExpiresAt() time.Time { return time.Unix(c.Exp, 0) }

// EffectiveScopes возвращает отсортированную копию scopes.
func (c Claims) EffectiveScopes() []string {
	if len(c.Scopes) == 0 {
		return nil
	}
	out := make([]string, len(c.Scopes))
	copy(out, c.Scopes)
	slices.Sort(out)
	return out
}

//
// Контракты верификатора подписи/базовых временных полей.
//

type Verifier interface {
	Verify(ctx context.Context, rawToken string) (*Claims, error)
}

// AudienceChecker — проверка совпадения aud.
type AudienceChecker func(cl *Claims, want string) bool

func DefaultAudienceChecker(cl *Claims, want string) bool {
	return slices.Contains(cl.Audience, want)
}

//
// Валидация OBO-политики для downstream-сервисов.
//

type OBOValidateOptions struct {
	// Требуемая аудитория (обычно — текущий сервис). Обязательна.
	WantAudience string

	// Ожидаемый актёр (например, "api-gateway").
	// Пусто — значение не сопоставляем, но сам факт наличия act.sub обязателен.
	WantActor string

	// Допуск по времени (компенсация дрейфа часов, напр. 30–60s).
	Leeway time.Duration

	// Максимальный TTL токена (напр. ≤ 5 минут).
	MaxTTL time.Duration

	// Проверка PoP: ожидаемый x5t#S256 клиентского сертификата (base64url).
	// Если непустой — PoP обязателен.
	MTLSThumbprint string

	// Anti-replay: callback должен вернуть true, если такой jti уже встречался.
	SeenJTI func(string) bool

	// Требовать, чтобы scopes были непустыми.
	RequireScopes bool
}

// ValidateOBO — усиленная проверка OBO-токена.
func ValidateOBO(now time.Time, cl *Claims, opt OBOValidateOptions) error {
	// 0) sub = UUID
	if _, err := uuid.Parse(cl.Subject); err != nil {
		return ErrBadSubject
	}

	// 1) aud: ровно один и тот, который ожидаем
	if len(cl.Audience) != 1 || cl.Audience[0] != opt.WantAudience {
		return ErrAudMismatch
	}

	// 2) actor: обязателен; если WantActor задан — должен совпасть
	if cl.Act == nil || strings.TrimSpace(cl.Act.Sub) == "" {
		return ErrMissingActor
	}
	if opt.WantActor != "" && cl.Act.Sub != opt.WantActor {
		return ErrActorMismatch
	}

	// 3) время жизни: exp/iat + leeway
	n := now.Unix()
	leeway := opt.Leeway
	if leeway < 0 {
		leeway = 0
	}
	if cl.Exp <= n-int64(leeway.Seconds()) {
		return ErrExpired
	}
	if cl.Iat > n+int64(leeway.Seconds()) {
		return ErrIATInFuture
	}

	// 3.1) ограничение TTL
	if opt.MaxTTL > 0 && time.Unix(cl.Exp, 0).Sub(time.Unix(cl.Iat, 0)) > opt.MaxTTL {
		return ErrTTLTooLong
	}

	// 4) jti + anti-replay
	if strings.TrimSpace(cl.Jti) == "" {
		return ErrMissingJTI
	}
	if opt.SeenJTI != nil && opt.SeenJTI(cl.Jti) {
		return ErrReplay
	}

	// 5) mTLS PoP (строгое сравнение base64url-отпечатка)
	if opt.MTLSThumbprint != "" {
		if cl.Cnf == nil || cl.Cnf.X5tS256 != opt.MTLSThumbprint {
			return ErrMTLSBindingMismatch
		}
	}

	// 6) scopes не должны быть пустыми (если требуется)
	if opt.RequireScopes && len(cl.Scopes) == 0 {
		return ErrMissingScopes
	}

	return nil
}

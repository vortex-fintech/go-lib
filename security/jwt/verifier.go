// go-lib/security/jwt/claims.go
package jwt

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Sentinel errors (удобно матчить в вызывающем коде).
var (
	ErrNilClaims           = errors.New("jwt: nil claims")
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
	ErrWalletMismatch      = errors.New("jwt: wallet mismatch")
	ErrAZPMismatch         = errors.New("jwt: azp mismatch")
)

// Actor (RFC 8693) — кто обменял токен (обычно клиент-сервис, напр. "api-gateway").
type Actor struct {
	Sub string `json:"sub"`
}

// Cnf (RFC 7800) — привязка к клиентскому сертификату (PoP).
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
	Azp   string `json:"azp,omitempty"` // кто получил user access от SSO (напр. "vortex-web")
	Act   *Actor `json:"act,omitempty"` // кто обменял токен (напр. "api-gateway")
	Cnf   *Cnf   `json:"cnf,omitempty"` // mTLS PoP
	SrcTH string `json:"src_th,omitempty"`

	// Контекст аутентификации
	ACR string   `json:"acr,omitempty"`
	AMR []string `json:"amr,omitempty"`

	// Контекст запроса
	WalletID string `json:"wallet_id,omitempty"`
	DeviceID string `json:"device_id,omitempty"`
}

func (c Claims) ExpiresAt() time.Time { return time.Unix(c.Exp, 0) }

// EffectiveScopes — отсортированная копия scopes.
func (c Claims) EffectiveScopes() []string {
	if len(c.Scopes) == 0 {
		return nil
	}
	out := make([]string, len(c.Scopes))
	copy(out, c.Scopes)
	slices.Sort(out)
	return out
}

// HasScopes — required ⊆ Scopes.
func (c Claims) HasScopes(required ...string) bool {
	if len(required) == 0 {
		return true
	}
	set := make(map[string]struct{}, len(c.Scopes))
	for _, s := range c.Scopes {
		set[s] = struct{}{}
	}
	for _, r := range required {
		if _, ok := set[r]; !ok {
			return false
		}
	}
	return true
}

// Verifier — контракт верификации подписи/базовых временных полей.
type Verifier interface {
	Verify(ctx context.Context, rawToken string) (*Claims, error)
}

// AudienceChecker — проверка совпадения aud.
type AudienceChecker func(cl *Claims, want string) bool

func DefaultAudienceChecker(cl *Claims, want string) bool {
	return slices.Contains(cl.Audience, want)
}

// OBOValidateOptions — усиленная проверка OBO-токена.
type OBOValidateOptions struct {
	WantAudience string   // обязательна
	WantActor    string   // если задан — act.sub должен совпасть
	WantWalletID string   // (опц.) cl.WalletID должен совпасть
	AllowedAZP   []string // (опц.) белый список azp (если список задан — azp обязателен)

	Leeway         time.Duration
	MaxTTL         time.Duration
	MTLSThumbprint string // если непустой — PoP обязателен
	SeenJTI        func(string) bool
	RequireScopes  bool
}

// ValidateOBO — строгая валидация OBO.
func ValidateOBO(now time.Time, cl *Claims, opt OBOValidateOptions) error {
	if cl == nil {
		return ErrNilClaims
	}

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

	// 2.1) (строгий) azp: если включён белый список — azp обязателен и должен быть в списке
	if len(opt.AllowedAZP) > 0 {
		if strings.TrimSpace(cl.Azp) == "" || !slices.Contains(opt.AllowedAZP, cl.Azp) {
			return ErrAZPMismatch
		}
	}

	// 3) время жизни: exp/iat + leeway
	leeway := max(opt.Leeway, 0)
	if now.Add(-leeway).After(time.Unix(cl.Exp, 0)) {
		return ErrExpired
	}
	if time.Unix(cl.Iat, 0).After(now.Add(leeway)) {
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

	// 7) (опц.) требуемый кошелёк
	if opt.WantWalletID != "" && cl.WalletID != opt.WantWalletID {
		return ErrWalletMismatch
	}

	return nil
}

// RequireScopes — ValidateOBO + проверка конкретных скоупов.
func RequireScopes(now time.Time, cl *Claims, opt OBOValidateOptions, required ...string) error {
	if err := ValidateOBO(now, cl, opt); err != nil {
		return err
	}
	if len(required) > 0 && !cl.HasScopes(required...) {
		return ErrMissingScopes
	}
	return nil
}

// RequireWallet — ValidateOBO + проверка wallet_id + скоупов.
func RequireWallet(now time.Time, cl *Claims, opt OBOValidateOptions, walletID string, required ...string) error {
	opt.WantWalletID = walletID
	return RequireScopes(now, cl, opt, required...)
}

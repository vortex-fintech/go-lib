package jwt

import (
	"context"
	"slices" // встроенный пакет начиная с Go 1.21
	"time"
)

// Claims — минимальный срез для внутренних OBO-токенов.
type Claims struct {
	Issuer   string   `json:"iss"` // SSO issuer, напр. https://sso.vortex.internal
	Subject  string   `json:"sub"` // user:<uuid>
	Audience []string `json:"aud"` // ровно один сервис: ["wallet"], ["auth"], ...
	Iat      int64    `json:"iat"` // время выпуска (unix seconds)
	Exp      int64    `json:"exp"` // время истечения (unix seconds)

	Sid string `json:"sid,omitempty"` // server session id, если был в access
	Jti string `json:"jti,omitempty"` // уникальный id токена (uuid)

	Scopes []string `json:"scope,omitempty"` // минимально-достаточные права для целевого сервиса
	Azp    string   `json:"azp,omitempty"`   // кто запросил exchange: "api-gateway" или "wallet"

	// Контекст аутентификации (если есть во внешнем access)
	ACR string   `json:"acr,omitempty"` // urn:vortex:acr:l1|l2|l3
	AMR []string `json:"amr,omitempty"` // ["pwd"], ["pwd","otp"], ...
}

func (c Claims) ExpiresAt() time.Time { return time.Unix(c.Exp, 0) }

// Verifier — без изменений.
type Verifier interface {
	Verify(ctx context.Context, rawToken string) (*Claims, error)
}

// AudienceChecker — проверка совпадения aud.
type AudienceChecker func(cl *Claims, want string) bool

func DefaultAudienceChecker(cl *Claims, want string) bool {
	return slices.Contains(cl.Audience, want)
}

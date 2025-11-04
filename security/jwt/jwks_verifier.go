// go-lib/security/jwt/verifier_jwks.go
package jwt

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// JWKS-клиент с in-memory кэшем + поддержкой Cache-Control/ETag.
type JWKSConfig struct {
	URL            string        // https://sso.internal/.well-known/jwks.json
	RefreshEvery   time.Duration // верхняя граница, если нет/большой max-age
	Timeout        time.Duration // HTTP timeout для JWKS-запроса
	ExpectedIssuer string        // опциональная проверка iss
	Leeway         time.Duration // опциональный leeway для iat/exp (если 0 => 5s)
}

type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
	Alg string `json:"alg"`
	Use string `json:"use"`
}

type jwks struct {
	Keys []jwk `json:"keys"`
}

type jwksVerifier struct {
	cfg         JWKSConfig
	mu          sync.RWMutex
	rsa         map[string]*rsa.PublicKey // kid -> key
	httpClient  *http.Client
	nextRefresh time.Time
	etag        string
}

func NewJWKSVerifier(cfg JWKSConfig) (Verifier, error) {
	tr := &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	v := &jwksVerifier{
		cfg: cfg,
		rsa: make(map[string]*rsa.PublicKey),
		httpClient: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: tr,
		},
	}
	if err := v.refresh(context.Background()); err != nil {
		return nil, err
	}
	return v, nil
}

func (v *jwksVerifier) Verify(ctx context.Context, raw string) (*Claims, error) {
	// мягкий refresh
	if time.Now().After(v.nextRefresh) {
		_ = v.refresh(ctx)
	}

	if l := len(raw); l == 0 || l > 16*1024 {
		return nil, errors.New("jwt: invalid size")
	}

	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return nil, errors.New("jwt: malformed")
	}

	// Header
	hdrJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	var hdr struct {
		Kid string `json:"kid"`
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(hdrJSON, &hdr); err != nil {
		return nil, err
	}
	if hdr.Kid == "" {
		return nil, errors.New("jwt: no kid")
	}
	// Разрешаем RS256 и PS256
	if hdr.Alg != "RS256" && hdr.Alg != "PS256" {
		return nil, errors.New("jwt: unexpected alg")
	}

	// Ключ по kid
	key, err := v.keyFor(ctx, hdr.Kid)
	if err != nil {
		return nil, err
	}

	signed := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, err
	}
	switch hdr.Alg {
	case "RS256":
		if err := verifyRS256(key, []byte(signed), sig); err != nil {
			return nil, err
		}
	case "PS256":
		if err := verifyPS256(key, []byte(signed), sig); err != nil {
			return nil, err
		}
	}

	// Payload
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	cl, err := decodeClaims(payload)
	if err != nil {
		return nil, err
	}

	// Time checks (leeway)
	leeway := v.cfg.Leeway
	if leeway <= 0 {
		leeway = 5 * time.Second
	}
	now := time.Now()
	if now.Add(-leeway).After(cl.ExpiresAt()) {
		return nil, errors.New("jwt: expired")
	}
	if cl.Iat > now.Add(leeway).Unix() {
		return nil, errors.New("jwt: iat in the future")
	}

	// Optional issuer check
	if v.cfg.ExpectedIssuer != "" && cl.Issuer != v.cfg.ExpectedIssuer {
		return nil, errors.New("jwt: unexpected iss")
	}

	return cl, nil
}

func (v *jwksVerifier) keyFor(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	k := v.rsa[kid]
	next := v.nextRefresh
	v.mu.RUnlock()

	if k != nil {
		return k, nil
	}
	if time.Now().After(next) {
		_ = v.refresh(ctx)
		v.mu.RLock()
		k = v.rsa[kid]
		v.mu.RUnlock()
		if k != nil {
			return k, nil
		}
	}
	return nil, errors.New("jwt: unknown kid")
}

func (v *jwksVerifier) refresh(ctx context.Context) error {
	if v.cfg.URL == "" {
		return errors.New("jwks: empty url")
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, v.cfg.URL, nil)
	if v.etag != "" {
		req.Header.Set("If-None-Match", v.etag)
	}
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// ok
	case http.StatusNotModified:
		v.mu.Lock()
		v.nextRefresh = time.Now().Add(v.refreshIntervalFromHeaders(resp.Header))
		v.mu.Unlock()
		return nil
	default:
		return fmt.Errorf("jwks: http %d", resp.StatusCode)
	}

	var set jwks
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return err
	}

	m := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, k := range set.Keys {
		if k.Kty != "RSA" {
			continue
		}
		if k.Use != "" && k.Use != "sig" {
			continue
		}
		if k.Alg != "" && k.Alg != "RS256" && k.Alg != "PS256" {
			continue
		}
		if k.Kid == "" || k.N == "" || k.E == "" {
			continue
		}

		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			return err
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			return err
		}
		var e int
		switch len(eBytes) {
		case 3:
			e = int(eBytes[0])<<16 | int(eBytes[1])<<8 | int(eBytes[2])
		case 1:
			e = int(eBytes[0])
		default:
			e = 65537
		}
		m[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}
	}

	v.mu.Lock()
	v.rsa = m
	v.etag = resp.Header.Get("ETag")
	v.nextRefresh = time.Now().Add(v.refreshIntervalFromHeaders(resp.Header))
	v.mu.Unlock()
	return nil
}

func (v *jwksVerifier) refreshIntervalFromHeaders(h http.Header) time.Duration {
	re := v.cfg.RefreshEvery
	if re <= 0 {
		re = 5 * time.Minute
	}
	if cc := h.Get("Cache-Control"); cc != "" {
		if d, ok := parseMaxAge(cc); ok && d > 0 && (re <= 0 || d < re) {
			re = d
		}
	}
	return re
}

func parseMaxAge(cc string) (time.Duration, bool) {
	for _, part := range strings.Split(cc, ",") {
		part = strings.TrimSpace(strings.ToLower(part))
		if strings.HasPrefix(part, "max-age=") {
			secStr := strings.TrimPrefix(part, "max-age=")
			if n, err := strconv.ParseInt(secStr, 10, 64); err == nil && n >= 0 {
				return time.Duration(n) * time.Second, true
			}
		}
	}
	return 0, false
}

// decodeClaims — БЕЗ legacy "scope": принимает только "scopes" как массив строк.
// Добавлена дедупликация scopes.
func decodeClaims(payload []byte) (*Claims, error) {
	type wire struct {
		Issuer   string   `json:"iss"`
		Subject  string   `json:"sub"`
		Audience any      `json:"aud"`
		Iat      int64    `json:"iat"`
		Exp      int64    `json:"exp"`
		Sid      string   `json:"sid,omitempty"`
		Jti      string   `json:"jti,omitempty"`
		Scopes   any      `json:"scopes,omitempty"`
		Azp      string   `json:"azp,omitempty"`
		ACR      string   `json:"acr,omitempty"`
		AMR      []string `json:"amr,omitempty"`
		Act      *Actor   `json:"act,omitempty"`
		Cnf      *Cnf     `json:"cnf,omitempty"`
		SrcTH    string   `json:"src_th,omitempty"`
		DeviceID string   `json:"device_id,omitempty"`
		WalletID string   `json:"wallet_id,omitempty"`
	}
	var w wire
	if err := json.Unmarshal(payload, &w); err != nil {
		return nil, err
	}

	cl := &Claims{
		Issuer:   w.Issuer,
		Subject:  w.Subject,
		Iat:      w.Iat,
		Exp:      w.Exp,
		Sid:      w.Sid,
		Jti:      w.Jti,
		Azp:      w.Azp,
		ACR:      w.ACR,
		AMR:      w.AMR,
		Act:      w.Act,
		Cnf:      w.Cnf,
		SrcTH:    w.SrcTH,
		DeviceID: w.DeviceID,
		WalletID: w.WalletID,
	}

	switch v := w.Audience.(type) {
	case string:
		if v != "" {
			cl.Audience = []string{v}
		}
	case []any:
		for _, it := range v {
			if s, ok := it.(string); ok && s != "" {
				cl.Audience = append(cl.Audience, s)
			}
		}
	case []string:
		cl.Audience = append(cl.Audience, v...)
	}

	// Дедупликация scopes
	appendUnique := func(s string, seen map[string]struct{}) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		cl.Scopes = append(cl.Scopes, s)
		seen[s] = struct{}{}
	}
	seen := make(map[string]struct{})

	switch v := w.Scopes.(type) {
	case nil:
		// ок
	case []string:
		for _, s := range v {
			appendUnique(s, seen)
		}
	case []any:
		for _, it := range v {
			if s, ok := it.(string); ok {
				appendUnique(s, seen)
			}
		}
	default:
		return nil, errors.New("jwt: scopes must be array of strings")
	}

	return cl, nil
}

func verifyRS256(pub *rsa.PublicKey, payload, sig []byte) error {
	h := sha256.Sum256(payload)
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, h[:], sig)
}

func verifyPS256(pub *rsa.PublicKey, payload, sig []byte) error {
	h := sha256.Sum256(payload)
	// Рекомендованный режим для JWT PS256: salt = len(hash)
	opts := &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash, Hash: crypto.SHA256}
	return rsa.VerifyPSS(pub, crypto.SHA256, h[:], sig, opts)
}

// X5tS256FromCert — x5t#S256 (base64url без паддинга) из DER-серта.
func X5tS256FromCert(cert *x509.Certificate) string {
	sum := sha256.Sum256(cert.Raw)
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

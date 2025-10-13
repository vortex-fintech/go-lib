// package jwt

package jwt

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Minimal JWKS client with in-memory cache.

type JWKSConfig struct {
	URL            string        // https://sso.internal/.well-known/jwks.json
	RefreshEvery   time.Duration // e.g. 5m
	Timeout        time.Duration // http timeout for JWKS fetch
	ExpectedIssuer string        // optional iss check
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
	if time.Now().After(v.nextRefresh) {
		_ = v.refresh(ctx)
	}

	if len(raw) == 0 || len(raw) > 16*1024 {
		return nil, errors.New("jwt: invalid size")
	}

	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return nil, errors.New("jwt: malformed")
	}

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
	if hdr.Alg != "" && hdr.Alg != "RS256" {
		return nil, errors.New("jwt: unexpected alg")
	}

	key, err := v.keyFor(ctx, hdr.Kid)
	if err != nil {
		return nil, err
	}

	signed := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, err
	}
	if err := verifyRS256(key, []byte(signed), sig); err != nil {
		return nil, err
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	cl, err := decodeClaims(payload)
	if err != nil {
		return nil, err
	}

	const leeway = 5 * time.Second
	if time.Now().Add(-leeway).After(cl.ExpiresAt()) {
		return nil, errors.New("jwt: expired")
	}

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
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
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
		if k.Alg != "" && k.Alg != "RS256" {
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
	re := v.cfg.RefreshEvery
	if re <= 0 {
		re = 5 * time.Minute
	}
	v.nextRefresh = time.Now().Add(re)
	v.mu.Unlock()
	return nil
}

// decodeClaims — tolerant к типам aud/scope (string | []string), совместим с твоим Claims.
func decodeClaims(payload []byte) (*Claims, error) {
	type wire struct {
		Issuer   string   `json:"iss"`
		Subject  string   `json:"sub"`
		Audience any      `json:"aud"`
		Iat      int64    `json:"iat"`
		Exp      int64    `json:"exp"`
		Sid      string   `json:"sid,omitempty"`
		Jti      string   `json:"jti,omitempty"`
		Scope    any      `json:"scope,omitempty"`
		Azp      string   `json:"azp,omitempty"`
		ACR      string   `json:"acr,omitempty"`
		AMR      []string `json:"amr,omitempty"`
	}
	var w wire
	if err := json.Unmarshal(payload, &w); err != nil {
		return nil, err
	}

	cl := &Claims{
		Issuer:  w.Issuer,
		Subject: w.Subject,
		Iat:     w.Iat,
		Exp:     w.Exp,
		Sid:     w.Sid,
		Jti:     w.Jti,
		Azp:     w.Azp,
		ACR:     w.ACR,
		AMR:     w.AMR,
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

	switch v := w.Scope.(type) {
	case string:
		if v != "" {
			cl.Scopes = strings.Fields(v)
		}
	case []any:
		for _, it := range v {
			if s, ok := it.(string); ok && s != "" {
				cl.Scopes = append(cl.Scopes, s)
			}
		}
	case []string:
		cl.Scopes = append(cl.Scopes, v...)
	}

	return cl, nil
}

// verifyRS256 — простая проверка подписи RSA-SHA256.
func verifyRS256(pub *rsa.PublicKey, payload, sig []byte) error {
	h := sha256.Sum256(payload)
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, h[:], sig)
}

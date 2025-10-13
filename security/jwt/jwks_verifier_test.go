package jwt

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- helpers ---

type jwksDoc struct {
	Keys []struct {
		Kty string `json:"kty"`
		Kid string `json:"kid"`
		N   string `json:"n"`
		E   string `json:"e"`
		Alg string `json:"alg"`
		Use string `json:"use"`
	} `json:"keys"`
}

func b64u(data []byte) string {
	return strings.TrimRight(base64.RawURLEncoding.EncodeToString(data), "=")
}

func makeRS256JWT(t *testing.T, kid string, priv *rsa.PrivateKey, payload map[string]any) string {
	t.Helper()
	hdr := map[string]any{
		"alg": "RS256",
		"typ": "JWT",
		"kid": kid,
	}
	hdrJSON, _ := json.Marshal(hdr)
	pldJSON, _ := json.Marshal(payload)
	h := b64u(hdrJSON) + "." + b64u(pldJSON)

	// sign
	sum := sha256.Sum256([]byte(h))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return h + "." + b64u(sig)
}

func jwksForKey(kid string, pub *rsa.PublicKey) []byte {
	n := b64u(pub.N.Bytes())
	// exponent to big-endian bytes (small e cases)
	var eBytes []byte
	switch {
	case pub.E < 256:
		eBytes = []byte{byte(pub.E)}
	case pub.E < 65536:
		eBytes = []byte{byte(pub.E >> 8), byte(pub.E)}
	default:
		eBytes = []byte{byte(pub.E >> 16), byte(pub.E >> 8), byte(pub.E)}
	}
	e := b64u(eBytes)

	doc := jwksDoc{
		Keys: []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
			Alg string `json:"alg"`
			Use string `json:"use"`
		}{
			{Kty: "RSA", Kid: kid, N: n, E: e, Alg: "RS256", Use: "sig"},
		},
	}
	out, _ := json.Marshal(doc)
	return out
}

// --- tests ---

func TestJWKSVerifier_BasicValid(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	pub := &priv.PublicKey
	kid := "k1"

	// JWKS server
	jwksBytes := jwksForKey(kid, pub)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(jwksBytes)
	}))
	defer ts.Close()

	v, err := NewJWKSVerifier(JWKSConfig{
		URL:          ts.URL,
		RefreshEvery: 1 * time.Minute,
		Timeout:      2 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewJWKSVerifier: %v", err)
	}

	now := time.Now().UTC()
	payload := map[string]any{
		"iss":   "https://sso.vortex.internal",
		"sub":   "user:123",
		"aud":   []string{"wallet"},
		"iat":   now.Unix(),
		"exp":   now.Add(2 * time.Minute).Unix(),
		"sid":   "sess:1",
		"jti":   "uuid-1",
		"scope": []string{"wallet:read"},
	}
	jwt := makeRS256JWT(t, kid, priv, payload)

	cl, err := v.Verify(context.Background(), jwt)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if cl.Subject != "user:123" || cl.Audience[0] != "wallet" {
		t.Fatalf("unexpected claims: %+v", cl)
	}
}

func TestJWKSVerifier_Expired(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	pub := &priv.PublicKey
	kid := "k1"
	jwksBytes := jwksForKey(kid, pub)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(jwksBytes)
	}))
	defer ts.Close()

	v, _ := NewJWKSVerifier(JWKSConfig{
		URL:          ts.URL,
		RefreshEvery: time.Minute,
		Timeout:      2 * time.Second,
	})

	now := time.Now().UTC()
	// истёк на 10с раньше; leeway у верификатора 5с → должно упасть
	payload := map[string]any{
		"iss": "https://sso.vortex.internal",
		"sub": "user:123",
		"aud": []string{"wallet"},
		"iat": now.Add(-2 * time.Minute).Unix(),
		"exp": now.Add(-10 * time.Second).Unix(),
	}
	j := makeRS256JWT(t, kid, priv, payload)
	if _, err := v.Verify(context.Background(), j); err == nil {
		t.Fatalf("expected expired error")
	}
}

func TestJWKSVerifier_UnexpectedAlg(t *testing.T) {
	// Cделаем заголовок с HS256 вручную (без подписи валидной — достаточно чтобы упало по alg)
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	pub := &priv.PublicKey
	kid := "k1"
	jwksBytes := jwksForKey(kid, pub)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(jwksBytes)
	}))
	defer ts.Close()

	v, _ := NewJWKSVerifier(JWKSConfig{
		URL:          ts.URL,
		RefreshEvery: time.Minute,
		Timeout:      2 * time.Second,
	})

	now := time.Now().UTC()
	hdr := map[string]any{"alg": "HS256", "typ": "JWT", "kid": kid}
	pld := map[string]any{
		"iss": "https://sso.vortex.internal",
		"sub": "user:1",
		"aud": []string{"wallet"},
		"iat": now.Unix(),
		"exp": now.Add(1 * time.Minute).Unix(),
	}
	hdrJSON, _ := json.Marshal(hdr)
	pldJSON, _ := json.Marshal(pld)
	token := b64u(hdrJSON) + "." + b64u(pldJSON) + "." + "bogus"

	if _, err := v.Verify(context.Background(), token); err == nil {
		t.Fatalf("expected unexpected alg error")
	}
}

func TestJWKSVerifier_UnknownKID(t *testing.T) {
	// JWKS с k1, токен с k2 → unknown kid
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	pub := &priv.PublicKey
	jwksBytes := jwksForKey("k1", pub)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(jwksBytes)
	}))
	defer ts.Close()

	v, _ := NewJWKSVerifier(JWKSConfig{
		URL:          ts.URL,
		RefreshEvery: time.Minute,
		Timeout:      2 * time.Second,
	})

	now := time.Now().UTC()
	j := makeRS256JWT(t, "k2", priv, map[string]any{
		"iss": "https://sso.vortex.internal",
		"sub": "user:1",
		"aud": []string{"wallet"},
		"iat": now.Unix(),
		"exp": now.Add(1 * time.Minute).Unix(),
	})
	if _, err := v.Verify(context.Background(), j); err == nil {
		t.Fatalf("expected unknown kid")
	}
}

func TestJWKSVerifier_ScopeStringNormalization(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	pub := &priv.PublicKey
	kid := "k1"
	jwksBytes := jwksForKey(kid, pub)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(jwksBytes)
	}))
	defer ts.Close()

	v, _ := NewJWKSVerifier(JWKSConfig{
		URL:          ts.URL,
		RefreshEvery: time.Minute,
		Timeout:      2 * time.Second,
	})

	now := time.Now().UTC()
	pl := map[string]any{
		"iss":   "https://sso.vortex.internal",
		"sub":   "user:1",
		"aud":   []string{"wallet"},
		"iat":   now.Unix(),
		"exp":   now.Add(1 * time.Minute).Unix(),
		"scope": "wallet:read wallet:transfer:create",
	}
	j := makeRS256JWT(t, kid, priv, pl)

	cl, err := v.Verify(context.Background(), j)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(cl.Scopes) != 2 || cl.Scopes[0] != "wallet:read" || cl.Scopes[1] != "wallet:transfer:create" {
		t.Fatalf("scope normalization failed: %+v", cl.Scopes)
	}
}

func TestJWKSVerifier_IssuerCheck(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	pub := &priv.PublicKey
	kid := "k1"
	jwksBytes := jwksForKey(kid, pub)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(jwksBytes)
	}))
	defer ts.Close()

	v, _ := NewJWKSVerifier(JWKSConfig{
		URL:            ts.URL,
		ExpectedIssuer: "https://sso.vortex.internal",
		RefreshEvery:   time.Minute,
		Timeout:        2 * time.Second,
	})

	now := time.Now().UTC()
	okTok := makeRS256JWT(t, kid, priv, map[string]any{
		"iss": "https://sso.vortex.internal",
		"sub": "user:1",
		"aud": []string{"wallet"},
		"iat": now.Unix(),
		"exp": now.Add(1 * time.Minute).Unix(),
	})
	if _, err := v.Verify(context.Background(), okTok); err != nil {
		t.Fatalf("issuer ok but failed: %v", err)
	}

	badTok := makeRS256JWT(t, kid, priv, map[string]any{
		"iss": "https://evil.example.com",
		"sub": "user:1",
		"aud": []string{"wallet"},
		"iat": now.Unix(),
		"exp": now.Add(1 * time.Minute).Unix(),
	})
	if _, err := v.Verify(context.Background(), badTok); err == nil {
		t.Fatalf("expected unexpected iss")
	}
}

func TestJWKSVerifier_SoftRefresh_ByTTL(t *testing.T) {
	// Стартуем с kid=k1, затем меняем JWKS на kid=k2 и ждём пока RefreshEvery истечёт.
	priv1, _ := rsa.GenerateKey(rand.Reader, 2048)
	pub1 := &priv1.PublicKey
	priv2, _ := rsa.GenerateKey(rand.Reader, 2048)
	pub2 := &priv2.PublicKey

	kid1 := "k1"
	kid2 := "k2"

	// переключаемый JWKS
	var current []byte
	current = jwksForKey(kid1, pub1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(current)
	}))
	defer ts.Close()

	v, _ := NewJWKSVerifier(JWKSConfig{
		URL:          ts.URL,
		RefreshEvery: 150 * time.Millisecond, // короткий TTL
		Timeout:      2 * time.Second,
	})

	now := time.Now().UTC()
	// Проверим k1
	j1 := makeRS256JWT(t, kid1, priv1, map[string]any{
		"iss": "https://sso.vortex.internal",
		"sub": "user:1",
		"aud": []string{"wallet"},
		"iat": now.Unix(),
		"exp": now.Add(1 * time.Minute).Unix(),
	})
	if _, err := v.Verify(context.Background(), j1); err != nil {
		t.Fatalf("k1 should verify: %v", err)
	}

	// Меняем JWKS на k2, ждём TTL и проверяем, что soft refresh подтянул k2
	current = jwksForKey(kid2, pub2)
	time.Sleep(200 * time.Millisecond) // больше RefreshEvery

	j2 := makeRS256JWT(t, kid2, priv2, map[string]any{
		"iss": "https://sso.vortex.internal",
		"sub": "user:2",
		"aud": []string{"wallet"},
		"iat": now.Unix(),
		"exp": now.Add(1 * time.Minute).Unix(),
	})
	if _, err := v.Verify(context.Background(), j2); err != nil {
		t.Fatalf("k2 should verify after refresh: %v", err)
	}
}

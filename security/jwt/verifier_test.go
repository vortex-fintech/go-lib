package jwt

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestValidateOBO_NilClaims(t *testing.T) {
	t.Parallel()

	err := ValidateOBO(time.Now(), nil, OBOValidateOptions{WantAudience: "wallet"})
	if err != ErrNilClaims {
		t.Fatalf("expected ErrNilClaims, got %v", err)
	}
}

func TestValidateOBO_BadSubject(t *testing.T) {
	t.Parallel()

	claims := &Claims{
		Subject:  "not-a-uuid",
		Audience: []string{"wallet"},
		Act:      &Actor{Sub: "api-gateway"},
		Jti:      "jti-123",
		Iat:      time.Now().Unix(),
		Exp:      time.Now().Add(time.Hour).Unix(),
	}

	err := ValidateOBO(time.Now(), claims, OBOValidateOptions{WantAudience: "wallet"})
	if err != ErrBadSubject {
		t.Fatalf("expected ErrBadSubject, got %v", err)
	}
}

func TestValidateOBO_AudMismatch(t *testing.T) {
	t.Parallel()

	claims := &Claims{
		Subject:  "550e8400-e29b-41d4-a716-446655440000",
		Audience: []string{"other"},
		Act:      &Actor{Sub: "api-gateway"},
		Jti:      "jti-123",
		Iat:      time.Now().Unix(),
		Exp:      time.Now().Add(time.Hour).Unix(),
	}

	err := ValidateOBO(time.Now(), claims, OBOValidateOptions{WantAudience: "wallet"})
	if err != ErrAudMismatch {
		t.Fatalf("expected ErrAudMismatch, got %v", err)
	}
}

func TestValidateOBO_MissingActor(t *testing.T) {
	t.Parallel()

	claims := &Claims{
		Subject:  "550e8400-e29b-41d4-a716-446655440000",
		Audience: []string{"wallet"},
		Jti:      "jti-123",
		Iat:      time.Now().Unix(),
		Exp:      time.Now().Add(time.Hour).Unix(),
	}

	err := ValidateOBO(time.Now(), claims, OBOValidateOptions{WantAudience: "wallet"})
	if err != ErrMissingActor {
		t.Fatalf("expected ErrMissingActor, got %v", err)
	}
}

func TestValidateOBO_ActorMismatch(t *testing.T) {
	t.Parallel()

	claims := &Claims{
		Subject:  "550e8400-e29b-41d4-a716-446655440000",
		Audience: []string{"wallet"},
		Act:      &Actor{Sub: "other-gateway"},
		Jti:      "jti-123",
		Iat:      time.Now().Unix(),
		Exp:      time.Now().Add(time.Hour).Unix(),
	}

	err := ValidateOBO(time.Now(), claims, OBOValidateOptions{
		WantAudience: "wallet",
		WantActor:    "api-gateway",
	})
	if err != ErrActorMismatch {
		t.Fatalf("expected ErrActorMismatch, got %v", err)
	}
}

func TestValidateOBO_AZPMismatch(t *testing.T) {
	t.Parallel()

	claims := &Claims{
		Subject:  "550e8400-e29b-41d4-a716-446655440000",
		Audience: []string{"wallet"},
		Act:      &Actor{Sub: "api-gateway"},
		Azp:      "unknown-client",
		Jti:      "jti-123",
		Iat:      time.Now().Unix(),
		Exp:      time.Now().Add(time.Hour).Unix(),
	}

	err := ValidateOBO(time.Now(), claims, OBOValidateOptions{
		WantAudience: "wallet",
		AllowedAZP:   []string{"vortex-web", "mobile-app"},
	})
	if err != ErrAZPMismatch {
		t.Fatalf("expected ErrAZPMismatch, got %v", err)
	}
}

func TestValidateOBO_Expired(t *testing.T) {
	t.Parallel()

	claims := &Claims{
		Subject:  "550e8400-e29b-41d4-a716-446655440000",
		Audience: []string{"wallet"},
		Act:      &Actor{Sub: "api-gateway"},
		Jti:      "jti-123",
		Iat:      time.Now().Add(-2 * time.Hour).Unix(),
		Exp:      time.Now().Add(-time.Hour).Unix(),
	}

	err := ValidateOBO(time.Now(), claims, OBOValidateOptions{WantAudience: "wallet"})
	if err != ErrExpired {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
}

func TestValidateOBO_IATInFuture(t *testing.T) {
	t.Parallel()

	claims := &Claims{
		Subject:  "550e8400-e29b-41d4-a716-446655440000",
		Audience: []string{"wallet"},
		Act:      &Actor{Sub: "api-gateway"},
		Jti:      "jti-123",
		Iat:      time.Now().Add(time.Hour).Unix(),
		Exp:      time.Now().Add(2 * time.Hour).Unix(),
	}

	err := ValidateOBO(time.Now(), claims, OBOValidateOptions{WantAudience: "wallet"})
	if err != ErrIATInFuture {
		t.Fatalf("expected ErrIATInFuture, got %v", err)
	}
}

func TestValidateOBO_Leeway_SubSecondExp_NoRounding(t *testing.T) {
	t.Parallel()

	now := time.Unix(10, 200_000_000)
	claims := &Claims{
		Subject:  "550e8400-e29b-41d4-a716-446655440000",
		Audience: []string{"wallet"},
		Act:      &Actor{Sub: "api-gateway"},
		Jti:      "jti-123",
		Iat:      now.Add(-time.Minute).Unix(),
		Exp:      9,
	}

	err := ValidateOBO(now, claims, OBOValidateOptions{
		WantAudience: "wallet",
		Leeway:       1500 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("expected token to be valid with sub-second leeway, got %v", err)
	}
}

func TestValidateOBO_Leeway_SubSecondIAT_NoRounding(t *testing.T) {
	t.Parallel()

	now := time.Unix(10, 900_000_000)
	claims := &Claims{
		Subject:  "550e8400-e29b-41d4-a716-446655440000",
		Audience: []string{"wallet"},
		Act:      &Actor{Sub: "api-gateway"},
		Jti:      "jti-123",
		Iat:      12,
		Exp:      now.Add(time.Hour).Unix(),
	}

	err := ValidateOBO(now, claims, OBOValidateOptions{
		WantAudience: "wallet",
		Leeway:       1500 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("expected iat check to honor sub-second leeway without rounding, got %v", err)
	}
}

func TestValidateOBO_TTLTooLong(t *testing.T) {
	t.Parallel()

	claims := &Claims{
		Subject:  "550e8400-e29b-41d4-a716-446655440000",
		Audience: []string{"wallet"},
		Act:      &Actor{Sub: "api-gateway"},
		Jti:      "jti-123",
		Iat:      time.Now().Unix(),
		Exp:      time.Now().Add(2 * time.Hour).Unix(),
	}

	err := ValidateOBO(time.Now(), claims, OBOValidateOptions{
		WantAudience: "wallet",
		MaxTTL:       time.Hour,
	})
	if err != ErrTTLTooLong {
		t.Fatalf("expected ErrTTLTooLong, got %v", err)
	}
}

func TestValidateOBO_MissingJTI(t *testing.T) {
	t.Parallel()

	claims := &Claims{
		Subject:  "550e8400-e29b-41d4-a716-446655440000",
		Audience: []string{"wallet"},
		Act:      &Actor{Sub: "api-gateway"},
		Iat:      time.Now().Unix(),
		Exp:      time.Now().Add(time.Hour).Unix(),
	}

	err := ValidateOBO(time.Now(), claims, OBOValidateOptions{WantAudience: "wallet"})
	if err != ErrMissingJTI {
		t.Fatalf("expected ErrMissingJTI, got %v", err)
	}
}

func TestValidateOBO_Replay(t *testing.T) {
	t.Parallel()

	claims := &Claims{
		Subject:  "550e8400-e29b-41d4-a716-446655440000",
		Audience: []string{"wallet"},
		Act:      &Actor{Sub: "api-gateway"},
		Jti:      "jti-seen",
		Iat:      time.Now().Unix(),
		Exp:      time.Now().Add(time.Hour).Unix(),
	}

	seen := map[string]bool{"jti-seen": true}
	seenFunc := func(jti string) bool { return seen[jti] }

	err := ValidateOBO(time.Now(), claims, OBOValidateOptions{
		WantAudience: "wallet",
		SeenJTI:      seenFunc,
	})
	if err != ErrReplay {
		t.Fatalf("expected ErrReplay, got %v", err)
	}
}

func TestValidateOBO_MTLSBindingMismatch(t *testing.T) {
	t.Parallel()

	claims := &Claims{
		Subject:  "550e8400-e29b-41d4-a716-446655440000",
		Audience: []string{"wallet"},
		Act:      &Actor{Sub: "api-gateway"},
		Cnf:      &Cnf{X5tS256: "thumbprint-a"},
		Jti:      "jti-123",
		Iat:      time.Now().Unix(),
		Exp:      time.Now().Add(time.Hour).Unix(),
	}

	err := ValidateOBO(time.Now(), claims, OBOValidateOptions{
		WantAudience:   "wallet",
		MTLSThumbprint: "thumbprint-b",
	})
	if err != ErrMTLSBindingMismatch {
		t.Fatalf("expected ErrMTLSBindingMismatch, got %v", err)
	}
}

func TestValidateOBO_MissingScopes(t *testing.T) {
	t.Parallel()

	claims := &Claims{
		Subject:  "550e8400-e29b-41d4-a716-446655440000",
		Audience: []string{"wallet"},
		Act:      &Actor{Sub: "api-gateway"},
		Jti:      "jti-123",
		Iat:      time.Now().Unix(),
		Exp:      time.Now().Add(time.Hour).Unix(),
	}

	err := ValidateOBO(time.Now(), claims, OBOValidateOptions{
		WantAudience:  "wallet",
		RequireScopes: true,
	})
	if err != ErrMissingScopes {
		t.Fatalf("expected ErrMissingScopes, got %v", err)
	}
}

func TestValidateOBO_WalletMismatch(t *testing.T) {
	t.Parallel()

	claims := &Claims{
		Subject:  "550e8400-e29b-41d4-a716-446655440000",
		Audience: []string{"wallet"},
		Act:      &Actor{Sub: "api-gateway"},
		Jti:      "jti-123",
		WalletID: "wallet-a",
		Iat:      time.Now().Unix(),
		Exp:      time.Now().Add(time.Hour).Unix(),
	}

	err := ValidateOBO(time.Now(), claims, OBOValidateOptions{
		WantAudience: "wallet",
		WantWalletID: "wallet-b",
	})
	if err != ErrWalletMismatch {
		t.Fatalf("expected ErrWalletMismatch, got %v", err)
	}
}

func TestValidateOBO_OK(t *testing.T) {
	t.Parallel()

	claims := &Claims{
		Subject:  "550e8400-e29b-41d4-a716-446655440000",
		Audience: []string{"wallet"},
		Act:      &Actor{Sub: "api-gateway"},
		Azp:      "vortex-web",
		Jti:      "jti-123",
		Scopes:   []string{"wallet:read", "wallet:write"},
		WalletID: "wallet-1",
		Iat:      time.Now().Unix(),
		Exp:      time.Now().Add(time.Hour).Unix(),
	}

	err := ValidateOBO(time.Now(), claims, OBOValidateOptions{
		WantAudience:  "wallet",
		WantActor:     "api-gateway",
		AllowedAZP:    []string{"vortex-web"},
		MaxTTL:        2 * time.Hour,
		RequireScopes: true,
		WantWalletID:  "wallet-1",
	})
	if err != nil {
		t.Fatalf("expected OK, got %v", err)
	}
}

func TestRequireScopes_MissingScope(t *testing.T) {
	t.Parallel()

	claims := &Claims{
		Subject:  "550e8400-e29b-41d4-a716-446655440000",
		Audience: []string{"wallet"},
		Act:      &Actor{Sub: "api-gateway"},
		Jti:      "jti-123",
		Scopes:   []string{"wallet:read"},
		Iat:      time.Now().Unix(),
		Exp:      time.Now().Add(time.Hour).Unix(),
	}

	err := RequireScopes(time.Now(), claims, OBOValidateOptions{WantAudience: "wallet"}, "wallet:read", "wallet:write")
	if err != ErrMissingScopes {
		t.Fatalf("expected ErrMissingScopes, got %v", err)
	}
}

func TestRequireScopes_OK(t *testing.T) {
	t.Parallel()

	claims := &Claims{
		Subject:  "550e8400-e29b-41d4-a716-446655440000",
		Audience: []string{"wallet"},
		Act:      &Actor{Sub: "api-gateway"},
		Jti:      "jti-123",
		Scopes:   []string{"wallet:read", "wallet:write"},
		Iat:      time.Now().Unix(),
		Exp:      time.Now().Add(time.Hour).Unix(),
	}

	err := RequireScopes(time.Now(), claims, OBOValidateOptions{WantAudience: "wallet"}, "wallet:read")
	if err != nil {
		t.Fatalf("expected OK, got %v", err)
	}
}

func TestRequireWallet_OK(t *testing.T) {
	t.Parallel()

	claims := &Claims{
		Subject:  "550e8400-e29b-41d4-a716-446655440000",
		Audience: []string{"wallet"},
		Act:      &Actor{Sub: "api-gateway"},
		Jti:      "jti-123",
		Scopes:   []string{"wallet:read"},
		WalletID: "wallet-1",
		Iat:      time.Now().Unix(),
		Exp:      time.Now().Add(time.Hour).Unix(),
	}

	err := RequireWallet(time.Now(), claims, OBOValidateOptions{WantAudience: "wallet"}, "wallet-1", "wallet:read")
	if err != nil {
		t.Fatalf("expected OK, got %v", err)
	}
}

func TestClaims_HasScopes(t *testing.T) {
	t.Parallel()

	claims := &Claims{Scopes: []string{"a", "b", "c"}}

	if !claims.HasScopes("a", "b") {
		t.Fatal("expected HasScopes(a,b) = true")
	}
	if claims.HasScopes("a", "d") {
		t.Fatal("expected HasScopes(a,d) = false")
	}
	if !claims.HasScopes() {
		t.Fatal("expected HasScopes() = true")
	}
}

func TestClaims_EffectiveScopes(t *testing.T) {
	t.Parallel()

	t.Run("sorted", func(t *testing.T) {
		claims := &Claims{Scopes: []string{"c", "a", "b"}}
		got := claims.EffectiveScopes()
		want := []string{"a", "b", "c"}
		if len(got) != len(want) {
			t.Fatalf("expected %d, got %d", len(want), len(got))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("expected %s at %d, got %s", want[i], i, got[i])
			}
		}
	})

	t.Run("empty", func(t *testing.T) {
		claims := &Claims{}
		if got := claims.EffectiveScopes(); got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})
}

func TestJWKSVerifier_RefreshOnUnknownKID(t *testing.T) {
	t.Parallel()

	keyA, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate keyA: %v", err)
	}
	keyB, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate keyB: %v", err)
	}

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := atomic.AddInt32(&calls, 1)
		set := map[string]any{}
		if call == 1 {
			set["keys"] = []map[string]string{jwkFromKey("kid-a", &keyA.PublicKey)}
		} else {
			set["keys"] = []map[string]string{
				jwkFromKey("kid-a", &keyA.PublicKey),
				jwkFromKey("kid-b", &keyB.PublicKey),
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(set)
	}))
	defer srv.Close()

	v, err := NewJWKSVerifier(JWKSConfig{
		URL:          srv.URL,
		RefreshEvery: time.Hour,
		Timeout:      2 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewJWKSVerifier: %v", err)
	}

	raw, err := signedTokenRS256("kid-b", keyB)
	if err != nil {
		t.Fatalf("signedTokenRS256: %v", err)
	}

	if _, err := v.Verify(context.Background(), raw); err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if atomic.LoadInt32(&calls) < 2 {
		t.Fatalf("expected at least one refresh call on unknown kid")
	}
}

func TestJWKSVerifier_RefreshOnUnknownKID_NilContext(t *testing.T) {
	t.Parallel()

	keyA, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate keyA: %v", err)
	}
	keyB, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate keyB: %v", err)
	}

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := atomic.AddInt32(&calls, 1)
		set := map[string]any{}
		if call == 1 {
			set["keys"] = []map[string]string{jwkFromKey("kid-a", &keyA.PublicKey)}
		} else {
			set["keys"] = []map[string]string{
				jwkFromKey("kid-a", &keyA.PublicKey),
				jwkFromKey("kid-b", &keyB.PublicKey),
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(set)
	}))
	defer srv.Close()

	v, err := NewJWKSVerifier(JWKSConfig{
		URL:          srv.URL,
		RefreshEvery: time.Hour,
		Timeout:      2 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewJWKSVerifier: %v", err)
	}

	raw, err := signedTokenRS256("kid-b", keyB)
	if err != nil {
		t.Fatalf("signedTokenRS256: %v", err)
	}

	if _, err := v.Verify(nil, raw); err != nil {
		t.Fatalf("Verify(nil, raw) failed: %v", err)
	}
	if atomic.LoadInt32(&calls) < 2 {
		t.Fatalf("expected at least one refresh call on unknown kid")
	}
}

func TestJWKSVerifier_SkipsInvalidKeyEntries(t *testing.T) {
	t.Parallel()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		set := map[string]any{
			"keys": []map[string]string{
				{
					"kty": "RSA",
					"kid": "broken",
					"alg": "RS256",
					"use": "sig",
					"n":   "@@@",
					"e":   "AQAB",
				},
				jwkFromKey("kid-ok", &key.PublicKey),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(set)
	}))
	defer srv.Close()

	v, err := NewJWKSVerifier(JWKSConfig{
		URL:          srv.URL,
		RefreshEvery: time.Hour,
		Timeout:      2 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewJWKSVerifier: %v", err)
	}

	raw, err := signedTokenRS256("kid-ok", key)
	if err != nil {
		t.Fatalf("signedTokenRS256: %v", err)
	}

	if _, err := v.Verify(context.Background(), raw); err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
}

func TestJWKSVerifier_KeepPreviousKeysOnEmptyRefresh(t *testing.T) {
	t.Parallel()

	keyA, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate keyA: %v", err)
	}

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		if call == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"keys": []map[string]string{jwkFromKey("kid-a", &keyA.PublicKey)},
			})
			return
		}

		// Нет ни одного валидного RSA ключа: кэш не должен стираться.
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{{
				"kty": "EC",
				"kid": "ec-1",
				"alg": "ES256",
				"use": "sig",
			}},
		})
	}))
	defer srv.Close()

	v, err := NewJWKSVerifier(JWKSConfig{
		URL:          srv.URL,
		RefreshEvery: 20 * time.Millisecond,
		Timeout:      2 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewJWKSVerifier: %v", err)
	}

	raw, err := signedTokenRS256("kid-a", keyA)
	if err != nil {
		t.Fatalf("signedTokenRS256: %v", err)
	}

	if _, err := v.Verify(context.Background(), raw); err != nil {
		t.Fatalf("first Verify failed: %v", err)
	}

	time.Sleep(60 * time.Millisecond)

	if _, err := v.Verify(context.Background(), raw); err != nil {
		t.Fatalf("second Verify failed after refresh: %v", err)
	}
	if atomic.LoadInt32(&calls) < 2 {
		t.Fatalf("expected refresh to be called at least twice")
	}
}

func TestX5tS256FromCert_Nil(t *testing.T) {
	t.Parallel()

	if got := X5tS256FromCert(nil); got != "" {
		t.Fatalf("expected empty thumbprint for nil cert, got %q", got)
	}
}

func signedTokenRS256(kid string, key *rsa.PrivateKey) (string, error) {
	header := map[string]string{"alg": "RS256", "typ": "JWT", "kid": kid}
	payload := map[string]any{
		"iss": "issuer",
		"sub": "550e8400-e29b-41d4-a716-446655440000",
		"aud": []string{"wallet"},
		"iat": time.Now().Add(-time.Minute).Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	hb, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	pb, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	hEnc := base64.RawURLEncoding.EncodeToString(hb)
	pEnc := base64.RawURLEncoding.EncodeToString(pb)
	msg := hEnc + "." + pEnc
	h := sha256.Sum256([]byte(msg))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h[:])
	if err != nil {
		return "", err
	}

	return msg + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func jwkFromKey(kid string, pub *rsa.PublicKey) map[string]string {
	e := big.NewInt(int64(pub.E)).Bytes()
	if len(e) == 0 {
		e = []byte{1}
	}
	return map[string]string{
		"kty": "RSA",
		"kid": kid,
		"alg": "RS256",
		"use": "sig",
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(e),
	}
}

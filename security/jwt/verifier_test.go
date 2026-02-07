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

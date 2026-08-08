package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Aayush9029/aads/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

func writePrivateKey(t *testing.T, curve elliptic.Curve, pkcs8 bool) (*ecdsa.PrivateKey, string) {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	var der []byte
	blockType := "EC PRIVATE KEY"
	if pkcs8 {
		der, err = x509.MarshalPKCS8PrivateKey(privateKey)
		blockType = "PRIVATE KEY"
	} else {
		der, err = x509.MarshalECPrivateKey(privateKey)
	}
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "private-key.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der}), 0600); err != nil {
		t.Fatal(err)
	}
	return privateKey, path
}

func testConfig(path string) *config.Config {
	return &config.Config{ClientID: "client-id", TeamID: "team-id", KeyID: "key-id", OrgID: "42", PrivateKeyPath: path}
}

func TestClientSecretUsesES256AndShortLifetime(t *testing.T) {
	privateKey, path := writePrivateKey(t, elliptic.P256(), true)
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	source, err := NewTokenSource(testConfig(path), WithClock(func() time.Time { return now }))
	if err != nil {
		t.Fatal(err)
	}
	secret, err := source.ClientSecret()
	if err != nil {
		t.Fatal(err)
	}
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(secret, claims, func(token *jwt.Token) (any, error) {
		return &privateKey.PublicKey, nil
	}, jwt.WithValidMethods([]string{"ES256"}), jwt.WithTimeFunc(func() time.Time { return now }))
	if err != nil || !token.Valid {
		t.Fatalf("parse client secret: %v", err)
	}
	if token.Header["kid"] != "key-id" || claims.Issuer != "team-id" || claims.Subject != "client-id" {
		t.Fatalf("unexpected JWT header or claims: %#v %#v", token.Header, claims)
	}
	if len(claims.Audience) != 1 || claims.Audience[0] != defaultAudience {
		t.Fatalf("audience = %#v", claims.Audience)
	}
	if lifetime := claims.ExpiresAt.Sub(claims.IssuedAt.Time); lifetime != defaultClientSecretLifetime || lifetime >= maximumClientSecretLifetime {
		t.Fatalf("client secret lifetime = %s", lifetime)
	}
}

func TestNewTokenSourceAcceptsSEC1AndRejectsWrongCurve(t *testing.T) {
	_, sec1Path := writePrivateKey(t, elliptic.P256(), false)
	if _, err := NewTokenSource(testConfig(sec1Path)); err != nil {
		t.Fatalf("SEC1 key: %v", err)
	}
	_, wrongCurvePath := writePrivateKey(t, elliptic.P384(), true)
	if _, err := NewTokenSource(testConfig(wrongCurvePath)); err == nil {
		t.Fatal("expected P-384 key to be rejected")
	}
}

func TestTokenExchangeFormCacheAndRefresh(t *testing.T) {
	_, path := writePrivateKey(t, elliptic.P256(), true)
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		if request.Method != http.MethodPost || request.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("unexpected request: %s %s", request.Method, request.Header.Get("Content-Type"))
		}
		if err := request.ParseForm(); err != nil {
			t.Error(err)
		}
		want := url.Values{"grant_type": {"client_credentials"}, "client_id": {"client-id"}, "scope": {"searchadsorg"}}
		for name, values := range want {
			if request.Form.Get(name) != values[0] {
				t.Errorf("form %s = %q", name, request.Form.Get(name))
			}
		}
		if request.Form.Get("client_secret") == "" {
			t.Error("client_secret is empty")
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(map[string]any{"access_token": "token", "token_type": "Bearer", "expires_in": 3600})
	}))
	defer server.Close()

	source, err := NewTokenSource(testConfig(path), WithTokenURL(server.URL), WithClock(func() time.Time { return now }))
	if err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for range 20 {
		group.Add(1)
		go func() {
			defer group.Done()
			if token, err := source.Token(context.Background()); err != nil || token != "token" {
				t.Errorf("Token() = %q, %v", token, err)
			}
		}()
	}
	group.Wait()
	if got := calls.Load(); got != 1 {
		t.Fatalf("token exchange calls = %d", got)
	}

	now = now.Add(3541 * time.Second)
	if _, err := source.Token(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("token should refresh within buffer, calls = %d", got)
	}
	source.Invalidate()
	if _, err := source.Token(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("invalidated token should refresh, calls = %d", got)
	}
}

func TestTokenExchangeReturnsOAuthError(t *testing.T) {
	_, path := writePrivateKey(t, elliptic.P256(), true)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(writer).Encode(map[string]string{"error": "invalid_client", "error_description": "bad credentials"})
	}))
	defer server.Close()
	source, err := NewTokenSource(testConfig(path), WithTokenURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.Token(context.Background()); err == nil || err.Error() != "token exchange failed (400): invalid_client: bad credentials" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTokenHonorsCancelledContext(t *testing.T) {
	_, path := writePrivateKey(t, elliptic.P256(), true)
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	source, err := NewTokenSource(testConfig(path), WithTokenURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := source.Token(ctx); err == nil {
		t.Fatal("expected cancellation error")
	}
}

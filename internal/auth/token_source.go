package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Aayush9029/aads/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

const (
	defaultTokenURL             = "https://appleid.apple.com/auth/oauth2/token"
	defaultAudience             = "https://appleid.apple.com"
	defaultScope                = "searchadsorg"
	defaultClientSecretLifetime = 179 * 24 * time.Hour
	maximumClientSecretLifetime = 180 * 24 * time.Hour
	defaultRefreshBuffer        = 60 * time.Second
)

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// TokenSource creates Apple client secrets and caches exchanged access tokens.
type TokenSource struct {
	config               *config.Config
	privateKey           *ecdsa.PrivateKey
	httpClient           httpDoer
	tokenURL             string
	now                  func() time.Time
	clientSecretLifetime time.Duration
	refreshBuffer        time.Duration

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

// Option customizes a TokenSource, primarily to make network and time behavior
// deterministic in tests.
type Option func(*TokenSource) error

func WithHTTPClient(client httpDoer) Option {
	return func(source *TokenSource) error {
		if client == nil {
			return fmt.Errorf("HTTP client is nil")
		}
		source.httpClient = client
		return nil
	}
}

func WithTokenURL(tokenURL string) Option {
	return func(source *TokenSource) error {
		parsed, err := url.Parse(tokenURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("invalid token URL %q", tokenURL)
		}
		source.tokenURL = tokenURL
		return nil
	}
}

func WithClock(now func() time.Time) Option {
	return func(source *TokenSource) error {
		if now == nil {
			return fmt.Errorf("clock is nil")
		}
		source.now = now
		return nil
	}
}

func WithClientSecretLifetime(lifetime time.Duration) Option {
	return func(source *TokenSource) error {
		if lifetime <= 0 || lifetime >= maximumClientSecretLifetime {
			return fmt.Errorf("client secret lifetime must be positive and less than 180 days")
		}
		source.clientSecretLifetime = lifetime
		return nil
	}
}

func WithRefreshBuffer(buffer time.Duration) Option {
	return func(source *TokenSource) error {
		if buffer < 0 {
			return fmt.Errorf("refresh buffer cannot be negative")
		}
		source.refreshBuffer = buffer
		return nil
	}
}

// NewTokenSource reads and validates the configured ES256 private key.
func NewTokenSource(cfg *config.Config, options ...Option) (*TokenSource, error) {
	if err := cfg.ValidateAuth(); err != nil {
		return nil, err
	}
	keyData, err := os.ReadFile(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	privateKey, err := parsePrivateKey(keyData)
	if err != nil {
		return nil, err
	}

	source := &TokenSource{
		config:               cfg,
		privateKey:           privateKey,
		httpClient:           &http.Client{Timeout: 30 * time.Second},
		tokenURL:             defaultTokenURL,
		now:                  time.Now,
		clientSecretLifetime: defaultClientSecretLifetime,
		refreshBuffer:        defaultRefreshBuffer,
	}
	for _, option := range options {
		if err := option(source); err != nil {
			return nil, err
		}
	}
	return source, nil
}

func parsePrivateKey(data []byte) (*ecdsa.PrivateKey, error) {
	block, rest := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("private key is not PEM encoded")
	}
	if strings.TrimSpace(string(rest)) != "" {
		return nil, fmt.Errorf("private key contains unexpected trailing data")
	}

	var parsed any
	var err error
	switch block.Type {
	case "PRIVATE KEY":
		parsed, err = x509.ParsePKCS8PrivateKey(block.Bytes)
	case "EC PRIVATE KEY":
		parsed, err = x509.ParseECPrivateKey(block.Bytes)
	default:
		return nil, fmt.Errorf("unsupported private key PEM type %q", block.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	privateKey, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not ECDSA")
	}
	if privateKey.Curve != elliptic.P256() {
		return nil, fmt.Errorf("private key must use the P-256 curve for ES256")
	}
	if privateKey.D == nil || privateKey.X == nil || privateKey.Y == nil || !privateKey.Curve.IsOnCurve(privateKey.X, privateKey.Y) {
		return nil, fmt.Errorf("private key is invalid")
	}
	return privateKey, nil
}

// ClientSecret signs a short-lived ES256 JWT suitable for Apple's token
// exchange. Its lifetime is always less than Apple's 180-day limit.
func (source *TokenSource) ClientSecret() (string, error) {
	now := source.now()
	claims := jwt.RegisteredClaims{
		Issuer:    source.config.TeamID,
		Subject:   source.config.ClientID,
		Audience:  jwt.ClaimStrings{defaultAudience},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(source.clientSecretLifetime)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = source.config.KeyID
	return token.SignedString(source.privateKey)
}

// Token returns a cached access token or performs a serialized token exchange.
func (source *TokenSource) Token(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("context is nil")
	}
	source.mu.Lock()
	defer source.mu.Unlock()

	if source.accessToken != "" && source.now().Add(source.refreshBuffer).Before(source.expiresAt) {
		return source.accessToken, nil
	}
	return source.exchange(ctx)
}

func (source *TokenSource) exchange(ctx context.Context) (string, error) {
	clientSecret, err := source.ClientSecret()
	if err != nil {
		return "", fmt.Errorf("create client secret: %w", err)
	}
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {source.config.ClientID},
		"client_secret": {clientSecret},
		"scope":         {defaultScope},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, source.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := source.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("exchange access token: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", parseOAuthError(response.StatusCode, body)
	}

	var tokenResponse struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return "", fmt.Errorf("parse token response: %w", err)
	}
	if strings.TrimSpace(tokenResponse.AccessToken) == "" {
		return "", fmt.Errorf("token response is missing access_token")
	}
	if tokenResponse.ExpiresIn <= 0 {
		return "", fmt.Errorf("token response has invalid expires_in")
	}

	source.accessToken = tokenResponse.AccessToken
	source.expiresAt = source.now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second)
	return source.accessToken, nil
}

func parseOAuthError(statusCode int, body []byte) error {
	var response struct {
		Code        string `json:"error"`
		Description string `json:"error_description"`
	}
	if json.Unmarshal(body, &response) == nil && response.Code != "" {
		if response.Description != "" {
			return fmt.Errorf("token exchange failed (%d): %s: %s", statusCode, response.Code, response.Description)
		}
		return fmt.Errorf("token exchange failed (%d): %s", statusCode, response.Code)
	}
	return fmt.Errorf("token exchange failed (%d): %s", statusCode, strings.TrimSpace(string(body)))
}

// Invalidate removes the cached access token. Client secrets are never cached.
func (source *TokenSource) Invalidate() {
	source.mu.Lock()
	defer source.mu.Unlock()
	source.accessToken = ""
	source.expiresAt = time.Time{}
}

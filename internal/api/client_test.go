package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Aayush9029/aads/internal/config"
)

type fakeTokenSource struct {
	mu          sync.Mutex
	generation  int
	invalidates int
}

func (source *fakeTokenSource) Token(context.Context) (string, error) {
	source.mu.Lock()
	defer source.mu.Unlock()
	return fmt.Sprintf("token-%d", source.generation), nil
}

func (source *fakeTokenSource) Invalidate() {
	source.mu.Lock()
	defer source.mu.Unlock()
	source.generation++
	source.invalidates++
}

func newTestClient(t *testing.T, server *httptest.Server, token TokenSource, options ...Option) *Client {
	t.Helper()
	allOptions := []Option{WithBaseURL(server.URL + "/api/v5"), WithHTTPClient(server.Client()), WithTokenSource(token)}
	allOptions = append(allOptions, options...)
	client, err := NewClient(&config.Config{OrgID: "123"}, allOptions...)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestRequestHeadersAndRawEnvelopePreservation(t *testing.T) {
	token := &fakeTokenSource{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v5/campaigns" {
			t.Errorf("path = %s", request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer token-0" {
			t.Errorf("authorization = %q", got)
		}
		if got := request.Header.Get("X-AP-Context"); got != "orgId=123" {
			t.Errorf("X-AP-Context = %q", got)
		}
		if got := request.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("content type = %q", got)
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.Write([]byte(`{"data":[{"id":1,"newField":"kept"}],"pagination":{"totalResults":1,"startIndex":0,"itemsPerPage":1,"cursor":"future"},"meta":{"future":true}}`))
	}))
	defer server.Close()
	client := newTestClient(t, server, token)

	response, err := client.Request(context.Background(), Request{
		Method:    http.MethodPost,
		Path:      "/campaigns",
		Body:      map[string]any{"name": "test"},
		Operation: OperationCreate,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(response.Body, []byte(`"newField":"kept"`)) {
		t.Fatalf("raw body lost response field: %s", response.Body)
	}
	if string(response.Envelope.Unknown["meta"]) != `{"future":true}` {
		t.Fatalf("unknown envelope fields = %s", response.Envelope.Unknown["meta"])
	}
	if string(response.Envelope.Pagination.Unknown["cursor"]) != `"future"` {
		t.Fatalf("unknown pagination = %s", response.Envelope.Pagination.Unknown["cursor"])
	}
}

func TestOrgContextIsOptionalForACLAndMe(t *testing.T) {
	var headers []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		headers = append(headers, request.Header.Get("X-AP-Context"))
		writer.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()
	client := newTestClient(t, server, &fakeTokenSource{})

	for _, path := range []string{"/acls", "/me?include=details"} {
		if _, err := client.Get(context.Background(), path); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := client.Request(context.Background(), Request{Method: http.MethodGet, Path: "/acls", OrgContext: OrgContextInclude}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"", "", "orgId=123"}; !reflect.DeepEqual(headers, want) {
		t.Fatalf("context headers = %#v, want %#v", headers, want)
	}
}

func TestUnauthorizedInvalidatesAndRetriesOnce(t *testing.T) {
	token := &fakeTokenSource{}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		if request.Header.Get("Authorization") == "Bearer token-0" {
			writer.WriteHeader(http.StatusUnauthorized)
			writer.Write([]byte(`{"error":{"errors":[{"messageCode":"UNAUTHORIZED","message":"expired"}]}}`))
			return
		}
		writer.Write([]byte(`{"data":{"id":1}}`))
	}))
	defer server.Close()
	client := newTestClient(t, server, token)

	if _, err := client.Request(context.Background(), Request{Method: http.MethodPost, Path: "/campaigns", Body: map[string]string{"name": "x"}, Operation: OperationCreate}); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 2 || token.invalidates != 1 {
		t.Fatalf("requests = %d, invalidates = %d", requests.Load(), token.invalidates)
	}
}

func TestUnauthorizedOnlyRefreshesOnce(t *testing.T) {
	token := &fakeTokenSource{}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		writer.WriteHeader(http.StatusUnauthorized)
		writer.Write([]byte(`{"errors":[{"message":"denied"}]}`))
	}))
	defer server.Close()
	client := newTestClient(t, server, token)

	_, err := client.Get(context.Background(), "/campaigns")
	var apiError *APIError
	if !errors.As(err, &apiError) || apiError.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected APIError 401, got %v", err)
	}
	if requests.Load() != 2 || token.invalidates != 1 {
		t.Fatalf("requests = %d, invalidates = %d", requests.Load(), token.invalidates)
	}
}

func TestRetryScheduleAndRetryAfter(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		attempt := requests.Add(1)
		if attempt == 1 {
			writer.Header().Set("Retry-After", "3")
			writer.WriteHeader(http.StatusTooManyRequests)
			return
		}
		if attempt == 2 {
			writer.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		writer.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()
	var waits []time.Duration
	client := newTestClient(t, server, &fakeTokenSource{}, WithSleeper(func(_ context.Context, duration time.Duration) error {
		waits = append(waits, duration)
		return nil
	}))

	if _, err := client.Get(context.Background(), "/campaigns"); err != nil {
		t.Fatal(err)
	}
	if want := []time.Duration{3 * time.Second, 4 * time.Second}; !reflect.DeepEqual(waits, want) {
		t.Fatalf("retry waits = %v, want %v", waits, want)
	}
}

func TestRetryAfterIsBounded(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			writer.Header().Set("Retry-After", "120")
			writer.WriteHeader(http.StatusTooManyRequests)
			return
		}
		writer.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()
	var wait time.Duration
	client := newTestClient(t, server, &fakeTokenSource{}, WithSleeper(func(_ context.Context, duration time.Duration) error {
		wait = duration
		return nil
	}))
	if _, err := client.Get(context.Background(), "/campaigns"); err != nil {
		t.Fatal(err)
	}
	if wait != maximumRetryWait {
		t.Fatalf("retry wait = %s, want %s", wait, maximumRetryWait)
	}
}

func TestAPIErrorWithEmptyBodyUsesStatusText(t *testing.T) {
	apiError := newAPIError(http.StatusBadGateway, nil, "")
	if got := apiError.Error(); got != "Apple Ads API error 502: Bad Gateway" {
		t.Fatalf("error = %q", got)
	}
}

func TestUnsafeCreatePostIsNotRetried(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		writer.Header().Set("Retry-After", "10")
		writer.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	client := newTestClient(t, server, &fakeTokenSource{}, WithSleeper(func(context.Context, time.Duration) error {
		t.Fatal("sleeper called for unsafe create")
		return nil
	}))

	_, err := client.Post(context.Background(), "/campaigns", map[string]string{"name": "x"}, OperationCreate)
	var apiError *APIError
	if !errors.As(err, &apiError) || apiError.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429 APIError, got %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("request count = %d", requests.Load())
	}
}

func TestFindPostCanRetryWhenExplicitlyMarked(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		writer.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()
	client := newTestClient(t, server, &fakeTokenSource{}, WithSleeper(func(context.Context, time.Duration) error { return nil }))

	if _, err := client.Post(context.Background(), "/search/geo", []byte(`[]`), OperationFind); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 2 {
		t.Fatalf("request count = %d", requests.Load())
	}
}

func TestStructuredAPIErrorAndRequestID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("X-Apple-Request-UUID", "request-123")
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte(`{"error":{"errors":[{"messageCode":"INVALID","message":"invalid value","field":"name","new":"kept"}]}}`))
	}))
	defer server.Close()
	client := newTestClient(t, server, &fakeTokenSource{})

	_, err := client.Get(context.Background(), "/campaigns")
	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("expected APIError, got %v", err)
	}
	if apiError.RequestID != "request-123" || len(apiError.Errors) != 1 || string(apiError.Errors[0].Unknown["new"]) != `"kept"` {
		t.Fatalf("unexpected API error: %#v", apiError)
	}
	if !strings.Contains(err.Error(), "invalid value (field: name)") || !strings.Contains(err.Error(), "request-123") {
		t.Fatalf("error text = %q", err)
	}
}

func TestVerboseOutputRedactsCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()
	var verbose bytes.Buffer
	client := newTestClient(t, server, &fakeTokenSource{}, WithVerbose(&verbose))
	if _, err := client.Get(context.Background(), "/campaigns"); err != nil {
		t.Fatal(err)
	}
	output := verbose.String()
	if strings.Contains(output, "token-0") || strings.Contains(output, "orgId=123") {
		t.Fatalf("verbose output leaked credentials: %s", output)
	}
	if strings.Count(output, "[redacted]") != 2 || !strings.Contains(output, "<- 200") {
		t.Fatalf("verbose output = %s", output)
	}
}

func TestRetrySleeperHonorsCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	client := newTestClient(t, server, &fakeTokenSource{}, WithSleeper(func(context.Context, time.Duration) error {
		cancel()
		return context.Canceled
	}))
	_, err := client.Get(ctx, "/campaigns")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestRequestRejectsInvalidPath(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	client := newTestClient(t, server, &fakeTokenSource{})
	if _, err := client.Get(context.Background(), "https://example.com/steal"); err == nil {
		t.Fatal("expected absolute URL to be rejected")
	}
}

func TestRetryAfterHTTPDateUsesInjectedClock(t *testing.T) {
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	if got := parseRetryAfter(now.Add(11*time.Second).Format(http.TimeFormat), now); got != 11*time.Second {
		t.Fatalf("retry after = %s", got)
	}
}

func TestEnvelopeDataRemainsValidJSON(t *testing.T) {
	var envelope Envelope
	if err := json.Unmarshal([]byte(`{"data":{"amount":"10.00"},"future":42}`), &envelope); err != nil {
		t.Fatal(err)
	}
	if string(envelope.Data) != `{"amount":"10.00"}` || string(envelope.Unknown["future"]) != "42" {
		t.Fatalf("envelope = %#v", envelope)
	}
}

func TestRawArrayResponseIsExposedAsData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Write([]byte(`[{"id":1,"displayName":"Toronto"}]`))
	}))
	defer server.Close()
	client := newTestClient(t, server, &fakeTokenSource{})
	response, err := client.Post(context.Background(), "/search/geo", []byte(`["Toronto"]`), OperationFind)
	if err != nil {
		t.Fatal(err)
	}
	if string(response.Envelope.Data) != `[{"id":1,"displayName":"Toronto"}]` {
		t.Fatalf("raw data = %s", response.Envelope.Data)
	}
}

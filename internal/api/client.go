package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Aayush9029/aads/internal/auth"
	"github.com/Aayush9029/aads/internal/config"
)

const defaultBaseURL = "https://api.searchads.apple.com/api/v5"

const maximumRetryWait = 16 * time.Second

var retrySchedule = [...]time.Duration{
	2 * time.Second,
	4 * time.Second,
	8 * time.Second,
	16 * time.Second,
}

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// TokenSource is the minimum authentication contract needed by Client.
type TokenSource interface {
	Token(context.Context) (string, error)
	Invalidate()
}

// Sleeper pauses a retry while remaining responsive to context cancellation.
type Sleeper func(context.Context, time.Duration) error

// Operation describes the semantics of a request when HTTP method alone is
// insufficient to decide whether replay is safe.
type Operation string

const (
	OperationAuto       Operation = ""
	OperationRead       Operation = "read"
	OperationFind       Operation = "find"
	OperationReport     Operation = "report"
	OperationCreate     Operation = "create"
	OperationUpdate     Operation = "update"
	OperationDelete     Operation = "delete"
	OperationBulkDelete Operation = "bulk-delete"
)

// OrgContextMode controls the X-AP-Context header. Auto includes the configured
// organization except for /acls and /me.
type OrgContextMode uint8

const (
	OrgContextAuto OrgContextMode = iota
	OrgContextInclude
	OrgContextOmit
)

// Request describes one API call. Body values are JSON encoded. []byte,
// json.RawMessage, and io.Reader bodies are buffered as already-encoded JSON.
type Request struct {
	Method     string
	Path       string
	Body       any
	Headers    http.Header
	Operation  Operation
	OrgContext OrgContextMode
}

// Response keeps both parsed envelope fields and the exact response bytes.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       json.RawMessage
	Envelope   Envelope
}

// Envelope is Apple Ads' generic response wrapper. Data and unknown top-level
// members remain raw so new API fields do not get discarded.
type Envelope struct {
	Data       json.RawMessage
	Pagination *PageDetail
	Error      *ErrorEnvelope
	Unknown    map[string]json.RawMessage
}

func (envelope *Envelope) UnmarshalJSON(data []byte) error {
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if raw, ok := fields["data"]; ok {
		envelope.Data = append(envelope.Data[:0], raw...)
		delete(fields, "data")
	}
	if raw, ok := fields["pagination"]; ok && string(raw) != "null" {
		envelope.Pagination = &PageDetail{}
		if err := json.Unmarshal(raw, envelope.Pagination); err != nil {
			return fmt.Errorf("parse pagination: %w", err)
		}
		delete(fields, "pagination")
	}
	if raw, ok := fields["error"]; ok && string(raw) != "null" {
		envelope.Error = &ErrorEnvelope{}
		if err := json.Unmarshal(raw, envelope.Error); err != nil {
			return fmt.Errorf("parse error envelope: %w", err)
		}
		delete(fields, "error")
	}
	envelope.Unknown = fields
	return nil
}

// PageDetail is the pagination member in Apple response envelopes.
type PageDetail struct {
	TotalResults  int                        `json:"totalResults"`
	StartIndex    int                        `json:"startIndex"`
	ItemsPerPage  int                        `json:"itemsPerPage"`
	Unknown       map[string]json.RawMessage `json:"-"`
	hasStartIndex bool
}

func (page *PageDetail) UnmarshalJSON(data []byte) error {
	type plain PageDetail
	if err := json.Unmarshal(data, (*plain)(page)); err != nil {
		return err
	}
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	_, page.hasStartIndex = fields["startIndex"]
	delete(fields, "totalResults")
	delete(fields, "startIndex")
	delete(fields, "itemsPerPage")
	page.Unknown = fields
	return nil
}

// Client is a context-aware Apple Ads API v5 transport.
type Client struct {
	httpClient httpDoer
	token      TokenSource
	baseURL    string
	orgID      string
	now        func() time.Time
	sleep      Sleeper
	verbose    io.Writer
}

// Option customizes Client dependencies.
type Option func(*Client) error

func WithHTTPClient(client httpDoer) Option {
	return func(apiClient *Client) error {
		if client == nil {
			return fmt.Errorf("HTTP client is nil")
		}
		apiClient.httpClient = client
		return nil
	}
}

func WithTokenSource(tokenSource TokenSource) Option {
	return func(apiClient *Client) error {
		if tokenSource == nil {
			return fmt.Errorf("token source is nil")
		}
		apiClient.token = tokenSource
		return nil
	}
}

func WithBaseURL(baseURL string) Option {
	return func(apiClient *Client) error {
		parsed, err := url.Parse(baseURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("invalid API base URL %q", baseURL)
		}
		apiClient.baseURL = strings.TrimRight(baseURL, "/")
		return nil
	}
}

func WithClock(now func() time.Time) Option {
	return func(apiClient *Client) error {
		if now == nil {
			return fmt.Errorf("clock is nil")
		}
		apiClient.now = now
		return nil
	}
}

func WithSleeper(sleeper Sleeper) Option {
	return func(apiClient *Client) error {
		if sleeper == nil {
			return fmt.Errorf("sleeper is nil")
		}
		apiClient.sleep = sleeper
		return nil
	}
}

// WithVerbose writes redacted request and retry diagnostics to writer.
func WithVerbose(writer io.Writer) Option {
	return func(apiClient *Client) error {
		if writer == nil {
			return fmt.Errorf("verbose writer is nil")
		}
		apiClient.verbose = writer
		return nil
	}
}

// NewClient creates a client using a 30-second HTTP timeout by default.
func NewClient(cfg *config.Config, options ...Option) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	client := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    defaultBaseURL,
		orgID:      cfg.OrgID,
		now:        time.Now,
		sleep:      sleepContext,
		verbose:    io.Discard,
	}
	for _, option := range options {
		if err := option(client); err != nil {
			return nil, err
		}
	}
	if client.token == nil {
		tokenSource, err := auth.NewTokenSource(cfg)
		if err != nil {
			return nil, fmt.Errorf("initialize authentication: %w", err)
		}
		client.token = tokenSource
	}
	return client, nil
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (client *Client) SetOrgID(orgID string) {
	client.orgID = orgID
}

func (client *Client) SetVerbose(enabled bool) {
	if enabled {
		client.verbose = os.Stderr
	} else {
		client.verbose = io.Discard
	}
}

func (client *Client) SetVerboseWriter(writer io.Writer) {
	if writer == nil {
		client.verbose = io.Discard
		return
	}
	client.verbose = writer
}

// Request executes a request and returns the raw and parsed response envelope.
func (client *Client) Request(ctx context.Context, specification Request) (*Response, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}
	method := strings.ToUpper(strings.TrimSpace(specification.Method))
	if method == "" {
		return nil, fmt.Errorf("HTTP method is required")
	}
	if !strings.HasPrefix(specification.Path, "/") || strings.HasPrefix(specification.Path, "//") {
		return nil, fmt.Errorf("API path must start with one slash")
	}
	body, err := encodeBody(specification.Body)
	if err != nil {
		return nil, err
	}

	authRefreshed := false
	retries := 0
	for {
		response, err := client.requestOnce(ctx, specification, method, body)
		if err == nil {
			return response, nil
		}

		apiError, isAPIError := err.(*APIError)
		if isAPIError && apiError.StatusCode == http.StatusUnauthorized && !authRefreshed {
			authRefreshed = true
			client.token.Invalidate()
			client.logf("authentication rejected; refreshing the access token\n")
			continue
		}

		if !isReplaySafe(method, specification.Path, specification.Operation) || retries >= len(retrySchedule) || !isRetryable(err) {
			return nil, err
		}
		wait := retrySchedule[retries]
		if isAPIError && apiError.RetryAfter > wait {
			wait = apiError.RetryAfter
		}
		if wait > maximumRetryWait {
			wait = maximumRetryWait
		}
		retries++
		client.logf("retrying in %s (%d/%d)\n", wait, retries, len(retrySchedule))
		if err := client.sleep(ctx, wait); err != nil {
			return nil, err
		}
	}
}

func encodeBody(body any) ([]byte, error) {
	switch value := body.(type) {
	case nil:
		return nil, nil
	case json.RawMessage:
		return append([]byte(nil), value...), nil
	case []byte:
		return append([]byte(nil), value...), nil
	case io.Reader:
		data, err := io.ReadAll(value)
		if err != nil {
			return nil, fmt.Errorf("read request body: %w", err)
		}
		return data, nil
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
		return data, nil
	}
}

func (client *Client) requestOnce(ctx context.Context, specification Request, method string, body []byte) (*Response, error) {
	accessToken, err := client.token.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}
	requestURL := client.baseURL + specification.Path
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	request, err := http.NewRequestWithContext(ctx, method, requestURL, reader)
	if err != nil {
		return nil, fmt.Errorf("create API request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+accessToken)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for name, values := range specification.Headers {
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}
	if client.includeOrgContext(specification) && client.orgID != "" {
		request.Header.Set("X-AP-Context", "orgId="+client.orgID)
	}

	client.logRequest(request)
	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, &transportError{cause: err}
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, &transportError{cause: fmt.Errorf("read API response: %w", err)}
	}
	client.logf("<- %d\n", response.StatusCode)

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		apiError := newAPIError(response.StatusCode, responseBody, requestID(response.Header))
		apiError.RetryAfter = parseRetryAfter(response.Header.Get("Retry-After"), client.now())
		return nil, apiError
	}

	result := &Response{
		StatusCode: response.StatusCode,
		Header:     response.Header.Clone(),
		Body:       append(json.RawMessage(nil), responseBody...),
	}
	trimmedBody := bytes.TrimSpace(responseBody)
	if len(trimmedBody) != 0 {
		if trimmedBody[0] == '{' {
			if err := json.Unmarshal(responseBody, &result.Envelope); err != nil {
				return nil, fmt.Errorf("parse API response envelope: %w", err)
			}
		} else {
			if !json.Valid(trimmedBody) {
				return nil, fmt.Errorf("parse API response: invalid JSON")
			}
			result.Envelope.Data = append(json.RawMessage(nil), trimmedBody...)
		}
	}
	return result, nil
}

func (client *Client) includeOrgContext(specification Request) bool {
	switch specification.OrgContext {
	case OrgContextInclude:
		return true
	case OrgContextOmit:
		return false
	default:
		path := specification.Path
		if query := strings.IndexByte(path, '?'); query >= 0 {
			path = path[:query]
		}
		return path != "/acls" && path != "/me"
	}
}

func (client *Client) logRequest(request *http.Request) {
	client.logf("-> %s %s\n", request.Method, request.URL.String())
	for name, values := range request.Header {
		value := strings.Join(values, ", ")
		if strings.EqualFold(name, "Authorization") || strings.EqualFold(name, "X-AP-Context") {
			value = "[redacted]"
		}
		client.logf("   %s: %s\n", name, value)
	}
}

func (client *Client) logf(format string, arguments ...any) {
	if client.verbose != nil {
		fmt.Fprintf(client.verbose, format, arguments...)
	}
}

type transportError struct {
	cause error
}

func (transportError *transportError) Error() string {
	return fmt.Sprintf("API request failed: %v", transportError.cause)
}

func (transportError *transportError) Unwrap() error {
	return transportError.cause
}

func isRetryable(err error) bool {
	if _, ok := err.(*transportError); ok {
		return true
	}
	apiError, ok := err.(*APIError)
	return ok && (apiError.StatusCode == http.StatusTooManyRequests || apiError.StatusCode >= 500)
}

func isReplaySafe(method, path string, operation Operation) bool {
	switch operation {
	case OperationRead, OperationFind, OperationReport, OperationUpdate, OperationDelete, OperationBulkDelete:
		return true
	case OperationCreate:
		return false
	}
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut, http.MethodDelete:
		return true
	case http.MethodPost:
		pathWithoutQuery := path
		if index := strings.IndexByte(pathWithoutQuery, '?'); index >= 0 {
			pathWithoutQuery = pathWithoutQuery[:index]
		}
		return strings.HasSuffix(pathWithoutQuery, "/find") ||
			strings.HasPrefix(pathWithoutQuery, "/reports/") ||
			strings.Contains(pathWithoutQuery, "/delete/bulk")
	default:
		return false
	}
}

func parseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	when, err := http.ParseTime(value)
	if err != nil || !when.After(now) {
		return 0
	}
	return when.Sub(now)
}

func requestID(header http.Header) string {
	for _, name := range []string{"X-Apple-Request-UUID", "X-Request-ID", "X-AP-Request-ID"} {
		if value := header.Get(name); value != "" {
			return value
		}
	}
	return ""
}

func (client *Client) Get(ctx context.Context, path string) (*Response, error) {
	return client.Request(ctx, Request{Method: http.MethodGet, Path: path, Operation: OperationRead})
}

func (client *Client) Post(ctx context.Context, path string, body any, operation Operation) (*Response, error) {
	return client.Request(ctx, Request{Method: http.MethodPost, Path: path, Body: body, Operation: operation})
}

func (client *Client) Put(ctx context.Context, path string, body any) (*Response, error) {
	return client.Request(ctx, Request{Method: http.MethodPut, Path: path, Body: body, Operation: OperationUpdate})
}

func (client *Client) Delete(ctx context.Context, path string) (*Response, error) {
	return client.Request(ctx, Request{Method: http.MethodDelete, Path: path, Operation: OperationDelete})
}

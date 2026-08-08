package cli

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Aayush9029/aads/internal/commands"
	"github.com/Aayush9029/aads/internal/config"
)

type recordingClient struct {
	requests []commands.Request
	result   commands.Result
	err      error
}

func (client *recordingClient) Do(_ context.Context, request commands.Request) (commands.Result, error) {
	client.requests = append(client.requests, request)
	return client.result, client.err
}

type cliHarness struct {
	stdin     io.Reader
	terminal  bool
	stdout    bytes.Buffer
	stderr    bytes.Buffer
	config    *config.Config
	loadErr   error
	saved     *config.Config
	saveErr   error
	client    commands.Client
	version   string
	loadCalls int
	saveCalls int
}

func newHarness() *cliHarness {
	return &cliHarness{
		stdin:   strings.NewReader(""),
		config:  &config.Config{OrgID: "123", DefaultCurrency: "USD"},
		client:  &recordingClient{result: commands.Result{Data: json.RawMessage(`[{"id":1,"name":"One"}]`)}},
		version: "1.2.3",
	}
}

func (harness *cliHarness) dependencies() Dependencies {
	return Dependencies{
		Stdin:      harness.stdin,
		Stdout:     &harness.stdout,
		Stderr:     &harness.stderr,
		Version:    harness.version,
		Client:     harness.client,
		IsTerminal: func(io.Reader) bool { return harness.terminal },
		LoadConfig: func() (*config.Config, error) {
			harness.loadCalls++
			if harness.loadErr != nil {
				return nil, harness.loadErr
			}
			if harness.config == nil {
				return nil, nil
			}
			copy := *harness.config
			return &copy, nil
		},
		SaveConfig: func(cfg *config.Config) error {
			harness.saveCalls++
			if cfg != nil {
				copy := *cfg
				harness.saved = &copy
			}
			return harness.saveErr
		},
	}
}

func (harness *cliHarness) run(args ...string) int {
	return Run(context.Background(), args, harness.dependencies())
}

func TestIsTerminalRejectsPipe(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	if isTerminal(reader) {
		t.Fatal("pipe was detected as a terminal")
	}
}

func TestRootHelpVersionAndDispatchErrors(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantCode    int
		wantStdout  string
		wantStderr  string
		configLoads int
		clientCalls int
	}{
		{name: "root help", args: []string{"--help"}, wantCode: 0, wantStdout: "Usage:\n  aads <command> <action> [flags]"},
		{name: "short root help", args: []string{"-h"}, wantCode: 0, wantStdout: "Apple Ads Campaign Management API v5 client"},
		{name: "missing command", wantCode: 2, wantStderr: "Usage:\n  aads <command> <action> [flags]"},
		{name: "version flag", args: []string{"--version"}, wantCode: 0, wantStdout: "aads 1.2.3\n"},
		{name: "version flag after tokens", args: []string{"campaigns", "list", "--version"}, wantCode: 0, wantStdout: "aads 1.2.3\n"},
		{name: "version command", args: []string{"version"}, wantCode: 0, wantStdout: "aads 1.2.3\n"},
		{name: "version arguments", args: []string{"version", "extra"}, wantCode: 2, wantStderr: "version does not accept arguments"},
		{name: "group help by omission", args: []string{"campaigns"}, wantCode: 0, wantStdout: "aads campaigns <action> [flags]"},
		{name: "group help flag", args: []string{"campaigns", "--help"}, wantCode: 0, wantStdout: "Actions:"},
		{name: "command help", args: []string{"campaigns", "get", "--help"}, wantCode: 0, wantStdout: "Get a campaign"},
		{name: "help command", args: []string{"help"}, wantCode: 0, wantStdout: "Apple Ads Campaign Management API v5 client"},
		{name: "help group", args: []string{"help", "campaigns"}, wantCode: 0, wantStdout: "aads campaigns <action> [flags]"},
		{name: "help action", args: []string{"help", "campaigns", "get"}, wantCode: 0, wantStdout: "Get a campaign"},
		{name: "group help action", args: []string{"campaigns", "help"}, wantCode: 0, wantStdout: "Actions:"},
		{name: "help too many arguments", args: []string{"help", "campaigns", "get", "extra"}, wantCode: 2, wantStderr: "help accepts a command and optional action"},
		{name: "unknown group", args: []string{"missing"}, wantCode: 2, wantStderr: `unknown command "missing"`},
		{name: "unknown action", args: []string{"campaigns", "missing"}, wantCode: 2, wantStderr: `unknown command "missing" for campaigns`},
		{name: "unknown command flag", args: []string{"campaigns", "list", "--bogus"}, wantCode: 2, wantStderr: "unknown flag --bogus for campaigns list"},
		{name: "unexpected argument", args: []string{"campaigns", "list", "stray"}, wantCode: 2, wantStderr: `unexpected argument "stray"`},
		{name: "duplicate flag", args: []string{"campaigns", "list", "--limit", "1", "--limit=2"}, wantCode: 2, wantStderr: "flag --limit was provided more than once"},
		{name: "missing global value", args: []string{"--output"}, wantCode: 2, wantStderr: "--output needs a value"},
		{name: "empty inline global value", args: []string{"--output=", "campaigns", "list"}, wantCode: 2, wantStderr: "--output needs a value"},
		{name: "invalid global bool", args: []string{"--envelope=maybe", "campaigns", "list"}, wantCode: 2, wantStderr: "--envelope must be true or false"},
		{name: "unsupported output", args: []string{"--output", "xml", "campaigns", "list"}, wantCode: 2, wantStderr: `unsupported output format "xml"`, configLoads: 0},
		{name: "unsupported partial fetch", args: []string{"--fields", "id,name", "campaigns", "create", "--name", "Test"}, wantCode: 2, wantStderr: "--fields is not supported by campaigns create"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			harness := newHarness()
			code := harness.run(test.args...)
			if code != test.wantCode {
				t.Fatalf("exit code = %d, want %d; stderr=%q", code, test.wantCode, harness.stderr.String())
			}
			if test.wantStdout != "" && !strings.Contains(harness.stdout.String(), test.wantStdout) {
				t.Errorf("stdout %q does not contain %q", harness.stdout.String(), test.wantStdout)
			}
			if test.wantStderr != "" && !strings.Contains(harness.stderr.String(), test.wantStderr) {
				t.Errorf("stderr %q does not contain %q", harness.stderr.String(), test.wantStderr)
			}
			if test.wantStdout != "" && test.wantStderr == "" && harness.stderr.Len() != 0 {
				t.Errorf("stderr = %q, want empty", harness.stderr.String())
			}
			if test.wantStderr != "" && test.wantStdout == "" && harness.stdout.Len() != 0 {
				t.Errorf("stdout = %q, want empty", harness.stdout.String())
			}
			if harness.loadCalls != test.configLoads {
				t.Errorf("config loads = %d, want %d", harness.loadCalls, test.configLoads)
			}
			client := harness.client.(*recordingClient)
			if len(client.requests) != test.clientCalls {
				t.Errorf("client calls = %d, want %d", len(client.requests), test.clientCalls)
			}
		})
	}
}

func TestGlobalFlagsWorkBeforeAndAfterCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "before", args: []string{"--output", "yaml", "--fields", "id,name", "campaigns", "list", "--limit", "2"}},
		{name: "after", args: []string{"campaigns", "list", "--limit", "2", "--fields=id,name", "--output=yaml"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			harness := newHarness()
			if code := harness.run(test.args...); code != 0 {
				t.Fatalf("exit code = %d; stderr=%q", code, harness.stderr.String())
			}
			client := harness.client.(*recordingClient)
			if len(client.requests) != 1 {
				t.Fatalf("client calls = %d, want 1", len(client.requests))
			}
			request := client.requests[0]
			if got := request.Query.Get("limit"); got != "2" {
				t.Errorf("limit = %q, want 2", got)
			}
			if got := request.Query.Get("fields"); got != "id,name" {
				t.Errorf("fields = %q, want id,name", got)
			}
			if !strings.Contains(harness.stdout.String(), "- id: 1") || strings.Contains(harness.stdout.String(), "{") {
				t.Errorf("stdout is not YAML data output: %q", harness.stdout.String())
			}
			if harness.stderr.Len() != 0 {
				t.Errorf("stderr = %q, want empty", harness.stderr.String())
			}
		})
	}
}

func TestJSONInputSourcesPreserveNumbers(t *testing.T) {
	temporary := t.TempDir()
	filePath := filepath.Join(temporary, "request.json")
	if err := os.WriteFile(filePath, []byte(`{"name":"file","budget":9007199254740993}`), 0600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		input string
		stdin string
		want  string
	}{
		{name: "inline", input: `{"name":"inline","budget":9007199254740993}`, want: "inline"},
		{name: "file", input: "@" + filePath, want: "file"},
		{name: "at stdin", input: "@-", stdin: `{"name":"at-stdin","budget":9007199254740993}`, want: "at-stdin"},
		{name: "dash stdin", input: "-", stdin: `{"name":"dash-stdin","budget":9007199254740993}`, want: "dash-stdin"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			harness := newHarness()
			harness.stdin = strings.NewReader(test.stdin)
			if code := harness.run("budgetorders", "create", "--from-json", test.input); code != 0 {
				t.Fatalf("exit code = %d; stderr=%q", code, harness.stderr.String())
			}
			client := harness.client.(*recordingClient)
			body, ok := client.requests[0].Body.(map[string]any)
			if !ok {
				t.Fatalf("body type = %T, want map", client.requests[0].Body)
			}
			if body["name"] != test.want {
				t.Errorf("name = %v, want %q", body["name"], test.want)
			}
			if amount, ok := body["budget"].(json.Number); !ok || amount.String() != "9007199254740993" {
				t.Errorf("budget = %#v (%T), want lossless json.Number", body["budget"], body["budget"])
			}
		})
	}
}

func TestJSONInputErrorsAreUsageErrors(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{name: "invalid", source: `{`, want: "parse JSON"},
		{name: "multiple values", source: `{} {}`, want: "parse JSON: multiple values"},
		{name: "missing file", source: "@/definitely/missing/aads.json", want: "read JSON"},
		{name: "empty object", source: `{}`, want: "--from-json cannot be empty"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			harness := newHarness()
			if code := harness.run("budgetorders", "create", "--from-json", test.source); code != 2 {
				t.Fatalf("exit code = %d, want 2; stderr=%q", code, harness.stderr.String())
			}
			if !strings.Contains(harness.stderr.String(), test.want) {
				t.Errorf("stderr %q does not contain %q", harness.stderr.String(), test.want)
			}
			if harness.stdout.Len() != 0 {
				t.Errorf("stdout = %q, want empty", harness.stdout.String())
			}
		})
	}
}

func TestOutputFormatsEnvelopeAndStreamSeparation(t *testing.T) {
	data := json.RawMessage(`[{"id":1,"name":"One"}]`)
	envelope := json.RawMessage(`{"data":[{"id":1,"name":"One"}],"pagination":{"totalResults":1}}`)
	tests := []struct {
		name      string
		args      []string
		want      []string
		doNotWant []string
	}{
		{name: "json data", args: []string{"campaigns", "list"}, want: []string{`"id": 1`, `"name": "One"`}, doNotWant: []string{"pagination", `"data"`}},
		{name: "json envelope", args: []string{"campaigns", "list", "--envelope"}, want: []string{`"data"`, `"pagination"`, `"totalResults": 1`}},
		{name: "yaml data", args: []string{"campaigns", "list", "--output", "yaml"}, want: []string{"- id: 1", "name: One"}, doNotWant: []string{"pagination:"}},
		{name: "table data", args: []string{"campaigns", "list", "-o", "table"}, want: []string{"ID", "NAME", "One"}, doNotWant: []string{"{"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			harness := newHarness()
			harness.client = &recordingClient{result: commands.Result{Data: data, Envelope: envelope}}
			if code := harness.run(test.args...); code != 0 {
				t.Fatalf("exit code = %d; stderr=%q", code, harness.stderr.String())
			}
			for _, value := range test.want {
				if !strings.Contains(harness.stdout.String(), value) {
					t.Errorf("stdout %q does not contain %q", harness.stdout.String(), value)
				}
			}
			for _, value := range test.doNotWant {
				if strings.Contains(harness.stdout.String(), value) {
					t.Errorf("stdout %q unexpectedly contains %q", harness.stdout.String(), value)
				}
			}
			if harness.stderr.Len() != 0 {
				t.Errorf("stderr = %q, want empty", harness.stderr.String())
			}
		})
	}
}

func TestClientAndOutputErrorsUseStderrAndExitOne(t *testing.T) {
	t.Run("client", func(t *testing.T) {
		harness := newHarness()
		harness.client = &recordingClient{err: errors.New("API unavailable")}
		if code := harness.run("campaigns", "list"); code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if harness.stdout.Len() != 0 || !strings.Contains(harness.stderr.String(), "aads: API unavailable") {
			t.Errorf("stdout=%q stderr=%q", harness.stdout.String(), harness.stderr.String())
		}
	})

	t.Run("output", func(t *testing.T) {
		harness := newHarness()
		harness.client = &recordingClient{result: commands.Result{Data: func() {}}}
		if code := harness.run("campaigns", "list"); code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if !strings.Contains(harness.stderr.String(), "unsupported type") {
			t.Errorf("stderr = %q", harness.stderr.String())
		}
	})
}

func TestDryRunRedactsHeadersAndDoesNotCallClient(t *testing.T) {
	harness := newHarness()
	harness.config.ClientID = "client-secret-looking-id"
	harness.config.OrgID = "777"
	harness.client = &recordingClient{}
	code := harness.run(
		"--org-id", "888",
		"keywords", "delete",
		"--campaign-id", "10",
		"--adgroup-id", "20",
		"--ids", "30,40",
		"--dry-run",
	)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, harness.stderr.String())
	}
	client := harness.client.(*recordingClient)
	if len(client.requests) != 0 {
		t.Fatalf("client calls = %d, want 0", len(client.requests))
	}
	var preview map[string]any
	if err := json.Unmarshal(harness.stdout.Bytes(), &preview); err != nil {
		t.Fatalf("decode preview: %v; output=%q", err, harness.stdout.String())
	}
	if preview["method"] != http.MethodPost || preview["path"] != "/campaigns/10/adgroups/20/targetingkeywords/delete/bulk" {
		t.Errorf("preview = %#v", preview)
	}
	headers, ok := preview["headers"].(map[string]any)
	if !ok {
		t.Fatalf("headers = %#v", preview["headers"])
	}
	if headers["Authorization"] != "Bearer [redacted]" || headers["X-AP-Context"] != "[redacted]" {
		t.Errorf("headers = %#v", headers)
	}
	if strings.Contains(harness.stdout.String(), harness.config.ClientID) {
		t.Error("dry-run leaked the client ID")
	}
	if harness.stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", harness.stderr.String())
	}
}

func TestDryRunDoesNotCallACLsToInferCurrency(t *testing.T) {
	harness := newHarness()
	harness.config.DefaultCurrency = ""
	harness.client = &recordingClient{result: commands.Result{Data: json.RawMessage(`[{"orgId":123,"currency":"USD"}]`)}}
	code := harness.run(
		"campaigns", "create",
		"--name", "Smoke",
		"--adam-id", "123456789",
		"--daily-budget", "10.00",
		"--countries", "US",
		"--ad-channel-type", "SEARCH",
		"--billing-event", "TAPS",
		"--supply-sources", "APPSTORE_SEARCH_RESULTS",
		"--dry-run",
	)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr=%q", code, harness.stderr.String())
	}
	if !strings.Contains(harness.stderr.String(), "dry-run needs --currency or a configured default") {
		t.Errorf("stderr = %q", harness.stderr.String())
	}
	if len(harness.client.(*recordingClient).requests) != 0 {
		t.Fatal("dry-run made an API request")
	}
}

func TestOrganizationOverrideValidation(t *testing.T) {
	for _, value := range []string{"0", "-1", "not-a-number"} {
		t.Run(value, func(t *testing.T) {
			harness := newHarness()
			harness.client = &recordingClient{}
			if code := harness.run("--org-id="+value, "campaigns", "list"); code != 2 {
				t.Fatalf("exit code = %d, want 2; stderr=%q", code, harness.stderr.String())
			}
			if !strings.Contains(harness.stderr.String(), "organization ID must be a positive integer") {
				t.Errorf("stderr = %q", harness.stderr.String())
			}
			if len(harness.client.(*recordingClient).requests) != 0 {
				t.Error("invalid organization ID reached the client")
			}
		})
	}
}

func TestConfiguredCurrencyAndGlobalOverrideBuildMoneyWithoutFloatConversion(t *testing.T) {
	tests := []struct {
		name         string
		configured   string
		overrideArgs []string
		want         string
	}{
		{name: "configured", configured: "cad", want: "CAD"},
		{name: "override", configured: "CAD", overrideArgs: []string{"--currency", "usd"}, want: "USD"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			harness := newHarness()
			harness.config.DefaultCurrency = test.configured
			args := append([]string{}, test.overrideArgs...)
			args = append(args,
				"campaigns", "create",
				"--name", "Launch",
				"--adam-id", "123456789",
				"--daily-budget", "10.01",
				"--countries", "CA",
				"--ad-channel-type", "SEARCH",
				"--billing-event", "TAPS",
				"--supply-sources", "APPSTORE_SEARCH_RESULTS",
			)
			if code := harness.run(args...); code != 0 {
				t.Fatalf("exit code = %d; stderr=%q", code, harness.stderr.String())
			}
			request := harness.client.(*recordingClient).requests[0]
			body := request.Body.(map[string]any)
			money := body["dailyBudgetAmount"].(map[string]any)
			if money["amount"] != "10.01" || money["currency"] != test.want {
				t.Errorf("dailyBudgetAmount = %#v, want exact amount and %s", money, test.want)
			}
		})
	}
}

func TestReportAllPagesUsesSelectorPagination(t *testing.T) {
	harness := newHarness()
	if code := harness.run(
		"reports", "campaigns",
		"--start-time", "2026-01-01",
		"--end-time", "2026-01-31",
		"--limit", "50",
		"--offset", "100",
		"--all",
	); code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, harness.stderr.String())
	}
	request := harness.client.(*recordingClient).requests[0]
	if !request.AllPages {
		t.Error("AllPages = false, want true")
	}
	if request.Query.Get("limit") != "" || request.Query.Get("offset") != "" {
		t.Errorf("report pagination leaked into query parameters: %v", request.Query)
	}
	body := request.Body.(map[string]any)
	selector := body["selector"].(map[string]any)
	pagination := selector["pagination"].(map[string]any)
	if pagination["limit"] != 50 || pagination["offset"] != 100 {
		t.Errorf("selector pagination = %#v", pagination)
	}
}

func TestCompleteReportJSONRequiresSelector(t *testing.T) {
	harness := newHarness()
	harness.client = &recordingClient{}
	code := harness.run(
		"reports", "campaigns",
		"--from-json", `{"startTime":"2026-01-01","endTime":"2026-01-31"}`,
		"--all",
	)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr=%q", code, harness.stderr.String())
	}
	if !strings.Contains(harness.stderr.String(), "reporting request must contain a selector object") {
		t.Errorf("stderr = %q", harness.stderr.String())
	}
	if len(harness.client.(*recordingClient).requests) != 0 {
		t.Error("invalid report reached the client")
	}
}

func TestDeleteConfirmationSafety(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		stdin      string
		terminal   bool
		wantCode   int
		wantCalls  int
		wantStdout string
		wantStderr string
	}{
		{name: "yes", args: []string{"campaigns", "delete", "--id", "10"}, stdin: "yes\n", terminal: true, wantCalls: 1, wantStdout: "{\n  \"status\": \"Campaign deleted\"\n}\n", wantStderr: "Delete a campaign? [y/N] "},
		{name: "single y", args: []string{"campaigns", "delete", "--id", "10"}, stdin: "Y\n", terminal: true, wantCalls: 1, wantStdout: "{\n  \"status\": \"Campaign deleted\"\n}\n", wantStderr: "Delete a campaign? [y/N] "},
		{name: "default no", args: []string{"campaigns", "delete", "--id", "10"}, stdin: "\n", terminal: true, wantCalls: 0, wantStderr: "Cancelled."},
		{name: "explicit no", args: []string{"campaigns", "delete", "--id", "10"}, stdin: "no\n", terminal: true, wantCalls: 0, wantStderr: "Cancelled."},
		{name: "non terminal", args: []string{"campaigns", "delete", "--id", "10"}, terminal: false, wantCode: 2, wantCalls: 0, wantStderr: "delete commands require --yes"},
		{name: "yes flag non terminal", args: []string{"campaigns", "delete", "--id", "10", "--yes"}, terminal: false, wantCalls: 1, wantStdout: "{\n  \"status\": \"Campaign deleted\"\n}\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			harness := newHarness()
			harness.stdin = strings.NewReader(test.stdin)
			harness.terminal = test.terminal
			harness.client = &recordingClient{}
			if code := harness.run(test.args...); code != test.wantCode {
				t.Fatalf("exit code = %d, want %d; stderr=%q", code, test.wantCode, harness.stderr.String())
			}
			client := harness.client.(*recordingClient)
			if len(client.requests) != test.wantCalls {
				t.Errorf("client calls = %d, want %d", len(client.requests), test.wantCalls)
			}
			if test.wantStdout != "" && harness.stdout.String() != test.wantStdout {
				t.Errorf("stdout = %q, want %q", harness.stdout.String(), test.wantStdout)
			}
			if test.wantStderr != "" && !strings.Contains(harness.stderr.String(), test.wantStderr) {
				t.Errorf("stderr %q does not contain %q", harness.stderr.String(), test.wantStderr)
			}
		})
	}
}

func TestConfigureLoadsPromptsAndSaves(t *testing.T) {
	harness := newHarness()
	harness.client = nil
	harness.config = &config.Config{ClientID: "old-client", TeamID: "old-team"}
	keyPath := validTestConfig(t).PrivateKeyPath
	harness.stdin = strings.NewReader("new-client\n\nnew-key\n987\n" + keyPath + "\ncad\n")
	if code := harness.run("configure"); code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, harness.stderr.String())
	}
	if harness.loadCalls != 1 || harness.saveCalls != 1 {
		t.Fatalf("load calls=%d save calls=%d, want 1 each", harness.loadCalls, harness.saveCalls)
	}
	want := config.Config{ClientID: "new-client", TeamID: "old-team", KeyID: "new-key", OrgID: "987", PrivateKeyPath: keyPath, DefaultCurrency: "CAD"}
	if harness.saved == nil || *harness.saved != want {
		t.Errorf("saved config = %#v, want %#v", harness.saved, want)
	}
	for _, prompt := range []string{"Client ID [old-client]: ", "Team ID [old-team]: ", "Key ID: ", "Organization ID: ", "Private key path: ", "Default currency: "} {
		if !strings.Contains(harness.stderr.String(), prompt) {
			t.Errorf("stderr %q does not contain prompt %q", harness.stderr.String(), prompt)
		}
	}
	if harness.stdout.String() != "Configuration saved.\n" {
		t.Errorf("stdout = %q", harness.stdout.String())
	}
}

func TestConfigureErrors(t *testing.T) {
	t.Run("load", func(t *testing.T) {
		harness := newHarness()
		harness.loadErr = errors.New("broken config")
		if code := harness.run("configure"); code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if !strings.Contains(harness.stderr.String(), "load config: broken config") || harness.saveCalls != 0 {
			t.Errorf("stderr=%q saveCalls=%d", harness.stderr.String(), harness.saveCalls)
		}
	})

	t.Run("input ends", func(t *testing.T) {
		harness := newHarness()
		harness.config = &config.Config{}
		harness.stdin = strings.NewReader("client\n")
		if code := harness.run("configure"); code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if !strings.Contains(harness.stderr.String(), "input ended") || harness.saveCalls != 0 {
			t.Errorf("stderr=%q saveCalls=%d", harness.stderr.String(), harness.saveCalls)
		}
	})

	t.Run("save", func(t *testing.T) {
		harness := newHarness()
		harness.config = &config.Config{}
		keyPath := validTestConfig(t).PrivateKeyPath
		harness.stdin = strings.NewReader("client\nteam\nkey\n123\n" + keyPath + "\nUSD\n")
		harness.saveErr = errors.New("disk full")
		if code := harness.run("configure"); code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if !strings.Contains(harness.stderr.String(), "disk full") || harness.stdout.Len() != 0 {
			t.Errorf("stdout=%q stderr=%q", harness.stdout.String(), harness.stderr.String())
		}
	})
}

func TestConfigureRejectsInvalidValuesBeforeSave(t *testing.T) {
	keyPath := validTestConfig(t).PrivateKeyPath
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "organization", input: "client\nteam\nkey\nnot-numeric\n" + keyPath + "\nUSD\n", want: "organization ID must be a positive integer"},
		{name: "currency", input: "client\nteam\nkey\n123\n" + keyPath + "\nUS\n", want: "default currency must be a three-letter ISO code"},
		{name: "auth", input: "client\nteam\nkey\n123\n/missing/private-key.pem\nUSD\n", want: "invalid config: read private key"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			harness := newHarness()
			harness.config = &config.Config{}
			harness.stdin = strings.NewReader(test.input)
			if code := harness.run("configure"); code != 1 {
				t.Fatalf("exit code = %d, want 1; stderr=%q", code, harness.stderr.String())
			}
			if !strings.Contains(harness.stderr.String(), test.want) {
				t.Errorf("stderr %q does not contain %q", harness.stderr.String(), test.want)
			}
			if harness.saveCalls != 0 || harness.stdout.Len() != 0 {
				t.Errorf("saveCalls=%d stdout=%q", harness.saveCalls, harness.stdout.String())
			}
		})
	}
}

func TestConfigLoadFailureForAPICommand(t *testing.T) {
	harness := newHarness()
	harness.loadErr = errors.New("bad YAML")
	if code := harness.run("campaigns", "list"); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(harness.stderr.String(), "load config: bad YAML") || harness.stdout.Len() != 0 {
		t.Errorf("stdout=%q stderr=%q", harness.stdout.String(), harness.stderr.String())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestRealClientUsesAuthOnlyForACLOrgOverrideAndVerboseRedaction(t *testing.T) {
	cfg := validTestConfig(t)
	cfg.OrgID = ""

	previousTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previousTransport })
	var apiRequests []*http.Request
	http.DefaultTransport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body := `{}`
		switch request.URL.Host {
		case "appleid.apple.com":
			body = `{"access_token":"top-secret-access-token","token_type":"Bearer","expires_in":3600}`
		case "api.searchads.apple.com":
			apiRequests = append(apiRequests, request.Clone(request.Context()))
			body = `{"data":[{"id":1}]}`
		default:
			return nil, fmt.Errorf("unexpected host %q", request.URL.Host)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
	})

	t.Run("ACL omits organization", func(t *testing.T) {
		harness := newHarness()
		harness.config = cfg
		harness.client = nil
		if code := harness.run("acls", "list"); code != 0 {
			t.Fatalf("exit code = %d; stderr=%q", code, harness.stderr.String())
		}
		if len(apiRequests) != 1 {
			t.Fatalf("API requests = %d, want 1", len(apiRequests))
		}
		if value := apiRequests[0].Header.Get("X-AP-Context"); value != "" {
			t.Errorf("X-AP-Context = %q, want omitted", value)
		}
	})

	apiRequests = nil
	t.Run("organization command rejects missing organization", func(t *testing.T) {
		harness := newHarness()
		harness.config = cfg
		harness.client = nil
		if code := harness.run("campaigns", "list"); code != 1 {
			t.Fatalf("exit code = %d, want 1", code)
		}
		if !strings.Contains(harness.stderr.String(), "org_id is required") || len(apiRequests) != 0 {
			t.Errorf("stderr=%q requests=%d", harness.stderr.String(), len(apiRequests))
		}
	})

	t.Run("override and verbose", func(t *testing.T) {
		harness := newHarness()
		harness.config = cfg
		harness.client = nil
		if code := harness.run("campaigns", "list", "--org-id", "456", "--verbose"); code != 0 {
			t.Fatalf("exit code = %d; stderr=%q", code, harness.stderr.String())
		}
		if len(apiRequests) != 1 {
			t.Fatalf("API requests = %d, want 1", len(apiRequests))
		}
		if value := apiRequests[0].Header.Get("X-AP-Context"); value != "orgId=456" {
			t.Errorf("X-AP-Context = %q, want orgId=456", value)
		}
		stderr := harness.stderr.String()
		for _, wanted := range []string{"-> GET https://api.searchads.apple.com/api/v5/campaigns", "Authorization: [redacted]", "X-Ap-Context: [redacted]", "<- 200"} {
			if !strings.Contains(stderr, wanted) {
				t.Errorf("stderr %q does not contain %q", stderr, wanted)
			}
		}
		if strings.Contains(stderr, "top-secret-access-token") || strings.Contains(stderr, "orgId=456") {
			t.Errorf("verbose output leaked a sensitive header: %q", stderr)
		}
		if !strings.Contains(harness.stdout.String(), `"id": 1`) {
			t.Errorf("stdout = %q", harness.stdout.String())
		}
	})
}

func validTestConfig(t *testing.T) *config.Config {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "private-key.pem")
	encoded := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	if err := os.WriteFile(path, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	return &config.Config{ClientID: "client", TeamID: "team", KeyID: "key", PrivateKeyPath: path, DefaultCurrency: "USD"}
}

func TestRequestSuccessTextOnlyForEmptyDeleteResponse(t *testing.T) {
	t.Run("empty result", func(t *testing.T) {
		harness := newHarness()
		harness.client = &recordingClient{}
		if code := harness.run("campaigns", "delete", "--id", "10", "--yes"); code != 0 {
			t.Fatalf("exit code = %d; stderr=%q", code, harness.stderr.String())
		}
		if harness.stdout.String() != "{\n  \"status\": \"Campaign deleted\"\n}\n" {
			t.Errorf("stdout = %q", harness.stdout.String())
		}
	})

	t.Run("response body wins", func(t *testing.T) {
		harness := newHarness()
		harness.client = &recordingClient{result: commands.Result{Data: json.RawMessage(`{"deleted":true}`)}}
		if code := harness.run("campaigns", "delete", "--id", "10", "--yes"); code != 0 {
			t.Fatalf("exit code = %d; stderr=%q", code, harness.stderr.String())
		}
		if strings.Contains(harness.stdout.String(), "Campaign deleted") || !strings.Contains(harness.stdout.String(), `"deleted": true`) {
			t.Errorf("stdout = %q", harness.stdout.String())
		}
	})
}

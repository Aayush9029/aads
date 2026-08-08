package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func clearConfigEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"AADS_CLIENT_ID",
		"AADS_TEAM_ID",
		"AADS_KEY_ID",
		"AADS_ORG_ID",
		"AADS_PRIVATE_KEY_PATH",
		"AADS_DEFAULT_CURRENCY",
		"AADS_CURRENCY",
	} {
		t.Setenv(name, "")
	}
}

func TestLoadFromAppliesEnvironmentPrecedence(t *testing.T) {
	clearConfigEnvironment(t)
	directory := t.TempDir()
	path := filepath.Join(directory, "config.yaml")
	contents := strings.Join([]string{
		"client_id: file-client",
		"team_id: file-team",
		"key_id: file-key",
		"org_id: file-org",
		"private_key_path: /file/key.pem",
		"default_currency: CAD",
	}, "\n")
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AADS_CLIENT_ID", "env-client")
	t.Setenv("AADS_ORG_ID", "env-org")
	t.Setenv("AADS_DEFAULT_CURRENCY", "USD")
	t.Setenv("AADS_CURRENCY", "EUR")

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ClientID != "env-client" || cfg.TeamID != "file-team" || cfg.OrgID != "env-org" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
	if cfg.DefaultCurrency != "EUR" {
		t.Fatalf("AADS_CURRENCY should have final precedence, got %q", cfg.DefaultCurrency)
	}
}

func TestLoadFromAllowsMissingFile(t *testing.T) {
	clearConfigEnvironment(t)
	t.Setenv("AADS_CLIENT_ID", "client")
	cfg, err := LoadFrom(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ClientID != "client" {
		t.Fatalf("client ID = %q", cfg.ClientID)
	}
}

func TestLoadFromRejectsUnknownFields(t *testing.T) {
	clearConfigEnvironment(t)
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("client_id: id\nsecret: should-not-be-here\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFrom(path); err == nil || !strings.Contains(err.Error(), "field secret not found") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}

func TestSaveToIsAtomicAndSecure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission semantics")
	}
	directory := filepath.Join(t.TempDir(), "aads")
	if err := os.Mkdir(directory, 0777); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "config.yaml")
	if err := os.WriteFile(path, []byte("old"), 0666); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{ClientID: "client", TeamID: "team", KeyID: "key", OrgID: "123", PrivateKeyPath: "/tmp/key.pem"}
	if err := SaveTo(path, cfg); err != nil {
		t.Fatal(err)
	}
	directoryInfo, err := os.Stat(directory)
	if err != nil {
		t.Fatal(err)
	}
	if got := directoryInfo.Mode().Perm(); got != 0700 {
		t.Fatalf("directory permissions = %o", got)
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fileInfo.Mode().Perm(); got != 0600 {
		t.Fatalf("file permissions = %o", got)
	}
	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ClientID != cfg.ClientID || loaded.PrivateKeyPath != cfg.PrivateKeyPath {
		t.Fatalf("saved config mismatch: %#v", loaded)
	}
	matches, err := filepath.Glob(filepath.Join(directory, ".config-*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary config left behind: %v", matches)
	}
}

func TestValidateAuthChecksRegularPrivateKey(t *testing.T) {
	cfg := &Config{ClientID: "client", TeamID: "team", KeyID: "key", PrivateKeyPath: t.TempDir()}
	if err := cfg.ValidateAuth(); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("expected regular file error, got %v", err)
	}
}

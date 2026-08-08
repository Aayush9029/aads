package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	directoryName = ".aads"
	fileName      = "config.yaml"
)

// Config contains persistent Apple Ads credentials and defaults. Secrets and
// access tokens are deliberately not part of this structure.
type Config struct {
	ClientID        string `yaml:"client_id"`
	TeamID          string `yaml:"team_id"`
	KeyID           string `yaml:"key_id"`
	OrgID           string `yaml:"org_id"`
	PrivateKeyPath  string `yaml:"private_key_path"`
	DefaultCurrency string `yaml:"default_currency,omitempty"`
}

// Path returns the default per-user configuration path.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, directoryName, fileName), nil
}

// Load reads the default configuration and applies AADS_* environment
// overrides. A missing configuration file is valid, allowing environment-only
// use in CI.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	return LoadFrom(path)
}

// LoadFrom reads configuration from path and applies environment overrides.
func LoadFrom(path string) (*Config, error) {
	cfg := &Config{}
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	if len(data) != 0 {
		decoder := yaml.NewDecoder(strings.NewReader(string(data)))
		decoder.KnownFields(true)
		if err := decoder.Decode(cfg); err != nil {
			return nil, fmt.Errorf("parse config %q: %w", path, err)
		}
	}

	applyEnvironment(cfg, os.LookupEnv)
	privateKeyPath, err := expandHome(cfg.PrivateKeyPath)
	if err != nil {
		return nil, err
	}
	cfg.PrivateKeyPath = privateKeyPath
	return cfg, nil
}

func applyEnvironment(cfg *Config, lookup func(string) (string, bool)) {
	overrides := []struct {
		name string
		set  func(string)
	}{
		{"AADS_CLIENT_ID", func(value string) { cfg.ClientID = value }},
		{"AADS_TEAM_ID", func(value string) { cfg.TeamID = value }},
		{"AADS_KEY_ID", func(value string) { cfg.KeyID = value }},
		{"AADS_ORG_ID", func(value string) { cfg.OrgID = value }},
		{"AADS_PRIVATE_KEY_PATH", func(value string) { cfg.PrivateKeyPath = value }},
		{"AADS_DEFAULT_CURRENCY", func(value string) { cfg.DefaultCurrency = value }},
		{"AADS_CURRENCY", func(value string) { cfg.DefaultCurrency = value }},
	}
	for _, override := range overrides {
		if value, ok := lookup(override.name); ok && value != "" {
			override.set(value)
		}
	}
}

func expandHome(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("expand private key path: %w", err)
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
}

// Save writes the default configuration atomically with user-only permissions.
func Save(cfg *Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	return SaveTo(path, cfg)
}

// SaveTo writes cfg atomically and enforces 0700 on its directory and 0600 on
// the file, including when either already exists with broader permissions.
func SaveTo(path string, cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if err := os.Chmod(directory, 0700); err != nil {
		return fmt.Errorf("secure config directory: %w", err)
	}

	temporary, err := os.CreateTemp(directory, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary config: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(0600); err != nil {
		temporary.Close()
		return fmt.Errorf("secure temporary config: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return fmt.Errorf("write temporary config: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync temporary config: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary config: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("secure config file: %w", err)
	}
	return syncDirectory(directory)
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open config directory: %w", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync config directory: %w", err)
	}
	return nil
}

// ValidateAuth checks the values required for an OAuth exchange.
func (cfg *Config) ValidateAuth() error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	required := []struct {
		name  string
		value string
	}{
		{"client_id", cfg.ClientID},
		{"team_id", cfg.TeamID},
		{"key_id", cfg.KeyID},
		{"private_key_path", cfg.PrivateKeyPath},
	}
	for _, field := range required {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("%s is required", field.name)
		}
	}
	info, err := os.Stat(cfg.PrivateKeyPath)
	if err != nil {
		return fmt.Errorf("read private key %q: %w", cfg.PrivateKeyPath, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("private key %q is not a regular file", cfg.PrivateKeyPath)
	}
	return nil
}

// Validate checks authentication values and the organization context.
func (cfg *Config) Validate() error {
	if err := cfg.ValidateAuth(); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.OrgID) == "" {
		return fmt.Errorf("org_id is required")
	}
	return nil
}

// WriteYAML writes a configuration without changing filesystem state. It is
// used by interactive command frontends that want to preview configuration.
func WriteYAML(writer io.Writer, cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if err := yaml.NewEncoder(writer).Encode(cfg); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	return nil
}

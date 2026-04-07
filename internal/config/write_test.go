package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beacon-stack/prism/internal/config"
)

func TestWriteConfigKey_ExplicitPath(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")

	path, err := config.WriteConfigKey(cfgFile, "auth.api_key", "test-key")
	if err != nil {
		t.Fatalf("WriteConfigKey() error = %v", err)
	}
	if path != cfgFile {
		t.Errorf("returned path = %q, want %q", path, cfgFile)
	}

	raw, err := os.ReadFile(cfgFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(raw), "test-key") {
		t.Errorf("config file does not contain written value; got:\n%s", raw)
	}
}

func TestWriteConfigKey_PreservesExistingKeys(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(cfgFile, []byte("server:\n  port: 9999\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := config.WriteConfigKey(cfgFile, "auth.api_key", "new-key"); err != nil {
		t.Fatalf("WriteConfigKey() error = %v", err)
	}

	raw, err := os.ReadFile(cfgFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	content := string(raw)
	if !strings.Contains(content, "9999") {
		t.Errorf("existing key 'server.port' was lost; got:\n%s", content)
	}
	if !strings.Contains(content, "new-key") {
		t.Errorf("new key 'auth.api_key' was not written; got:\n%s", content)
	}
}

// TestWriteConfigKey_RoundTrip verifies that a key written by WriteConfigKey
// can be read back by Load, confirming the write→load path is consistent.
// This is the core invariant that was broken in the Docker restart bug.
func TestWriteConfigKey_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")

	if _, err := config.WriteConfigKey(cfgFile, "auth.api_key", "stable-key"); err != nil {
		t.Fatalf("WriteConfigKey() error = %v", err)
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Auth.APIKey.Value() != "stable-key" {
		t.Errorf("Load() Auth.APIKey = %q, want %q", cfg.Auth.APIKey.Value(), "stable-key")
	}
}

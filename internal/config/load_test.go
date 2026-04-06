package config_test

import (
	"os"
	"testing"

	"github.com/beacon-media/prism/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Port != config.DefaultPort {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, config.DefaultPort)
	}
	if cfg.Log.Level != config.DefaultLogLevel {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, config.DefaultLogLevel)
	}
	if cfg.Database.Driver != config.DefaultDBDriver {
		t.Errorf("Database.Driver = %q, want %q", cfg.Database.Driver, config.DefaultDBDriver)
	}
}

func TestLoad_EnvVarOverride(t *testing.T) {
	t.Setenv("PRISM_AUTH_API_KEY", "my-secret-key")
	t.Setenv("PRISM_SERVER_PORT", "9999")
	t.Setenv("PRISM_LOG_LEVEL", "debug")

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Auth.APIKey.Value() != "my-secret-key" {
		t.Errorf("Auth.APIKey = %q, want %q", cfg.Auth.APIKey.Value(), "my-secret-key")
	}
	if cfg.Server.Port != 9999 {
		t.Errorf("Server.Port = %d, want 9999", cfg.Server.Port)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level = %q, want debug", cfg.Log.Level)
	}
}

func TestLoad_ConfigFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(`
server:
  port: 8888
auth:
  api_key: "file-key"
log:
  format: text
`)
	f.Close()

	cfg, err := config.Load(f.Name())
	if err != nil {
		t.Fatalf("Load(%q) error = %v", f.Name(), err)
	}

	if cfg.Server.Port != 8888 {
		t.Errorf("Server.Port = %d, want 8888", cfg.Server.Port)
	}
	if cfg.Auth.APIKey.Value() != "file-key" {
		t.Errorf("Auth.APIKey = %q, want %q", cfg.Auth.APIKey.Value(), "file-key")
	}
	if cfg.Log.Format != "text" {
		t.Errorf("Log.Format = %q, want text", cfg.Log.Format)
	}
}

func TestSecret_NeverLeaks(t *testing.T) {
	s := config.Secret("super-secret")

	if s.String() != "***" {
		t.Errorf("String() = %q, want ***", s.String())
	}
	if s.GoString() != "***" {
		t.Errorf("GoString() = %q, want ***", s.GoString())
	}

	b, err := s.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"***"` {
		t.Errorf("MarshalJSON() = %s, want \"***\"", b)
	}

	if s.Value() != "super-secret" {
		t.Errorf("Value() = %q, want super-secret", s.Value())
	}
}

func TestEnsureAPIKey_Generates(t *testing.T) {
	cfg := &config.Config{}

	generated, err := config.EnsureAPIKey(cfg)
	if err != nil {
		t.Fatalf("EnsureAPIKey() error = %v", err)
	}
	if !generated {
		t.Error("EnsureAPIKey() generated = false, want true")
	}
	if cfg.Auth.APIKey.IsEmpty() {
		t.Error("Auth.APIKey is empty after generation")
	}
	if len(cfg.Auth.APIKey.Value()) != 64 { // 32 bytes hex-encoded
		t.Errorf("generated key length = %d, want 64", len(cfg.Auth.APIKey.Value()))
	}
}

func TestEnsureAPIKey_NoOpWhenSet(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{APIKey: "existing-key"},
	}

	generated, err := config.EnsureAPIKey(cfg)
	if err != nil {
		t.Fatalf("EnsureAPIKey() error = %v", err)
	}
	if generated {
		t.Error("EnsureAPIKey() generated = true, want false when key exists")
	}
	if cfg.Auth.APIKey.Value() != "existing-key" {
		t.Errorf("Auth.APIKey changed, want existing-key got %q", cfg.Auth.APIKey.Value())
	}
}

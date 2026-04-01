package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sudores/twt-test-task/pkg/config"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(f, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return f
}

func TestLoad_Defaults(t *testing.T) {
	path := writeConfig(t, `
server:
  port: 9090
  read_timeout: 10s
  write_timeout: 20s
  idle_timeout: 30s
proxy:
  upstream_url: "https://example.com"
  timeout: 5s
rate_limit:
  enabled: true
  requests_per_second: 50
  burst: 100
log:
  level: "debug"
  pretty: false
`)

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("port: want 9090, got %d", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout.Duration != 10*time.Second {
		t.Errorf("read_timeout: want 10s, got %v", cfg.Server.ReadTimeout)
	}
	if cfg.Proxy.UpstreamURL != "https://example.com" {
		t.Errorf("upstream_url: want https://example.com, got %q", cfg.Proxy.UpstreamURL)
	}
	if cfg.Proxy.Timeout.Duration != 5*time.Second {
		t.Errorf("timeout: want 5s, got %v", cfg.Proxy.Timeout)
	}
	if !cfg.RateLimit.Enabled {
		t.Error("rate_limit.enabled: want true")
	}
	if cfg.RateLimit.RequestsPerSecond != 50 {
		t.Errorf("requests_per_second: want 50, got %v", cfg.RateLimit.RequestsPerSecond)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("log.level: want debug, got %q", cfg.Log.Level)
	}
}

func TestLoad_EnvSubstitution_VarSet(t *testing.T) {
	t.Setenv("TEST_PORT", "7070")
	t.Setenv("TEST_URL", "https://env.example.com")

	path := writeConfig(t, `
server:
  port: ${TEST_PORT:-8080}
proxy:
  upstream_url: "${TEST_URL:-https://default.com}"
  timeout: 5s
`)

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Port != 7070 {
		t.Errorf("port: want 7070, got %d", cfg.Server.Port)
	}
	if cfg.Proxy.UpstreamURL != "https://env.example.com" {
		t.Errorf("upstream_url: want https://env.example.com, got %q", cfg.Proxy.UpstreamURL)
	}
}

func TestLoad_EnvSubstitution_UsesDefault(t *testing.T) {
	os.Unsetenv("TEST_MISSING_VAR")

	path := writeConfig(t, `
server:
  port: ${TEST_MISSING_VAR:-1234}
proxy:
  upstream_url: "${TEST_MISSING_URL:-https://fallback.com}"
  timeout: 5s
`)

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Port != 1234 {
		t.Errorf("port: want 1234, got %d", cfg.Server.Port)
	}
	if cfg.Proxy.UpstreamURL != "https://fallback.com" {
		t.Errorf("upstream_url: want https://fallback.com, got %q", cfg.Proxy.UpstreamURL)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := config.Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeConfig(t, "server:\n  port: {unclosed\n")
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestLoad_InvalidDuration(t *testing.T) {
	path := writeConfig(t, `
proxy:
  upstream_url: "https://example.com"
  timeout: notaduration
`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for invalid duration, got nil")
	}
}

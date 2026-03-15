package mcp

import (
	"os"
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Clear any env vars that could interfere.
	for _, key := range []string{"STOA_MCP_API_URL", "STOA_MCP_API_KEY", "STOA_MCP_PORT", "STOA_MCP_BASE_URL"} {
		t.Setenv(key, "")
	}

	cfg := LoadConfig()

	if cfg.APIURL != "http://localhost:8080" {
		t.Errorf("APIURL = %q, want %q", cfg.APIURL, "http://localhost:8080")
	}
	if cfg.APIKey != "" {
		t.Errorf("APIKey = %q, want empty", cfg.APIKey)
	}
	if cfg.Port != 8090 {
		t.Errorf("Port = %d, want %d", cfg.Port, 8090)
	}
	if cfg.BaseURL != "http://localhost:8090" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "http://localhost:8090")
	}
}

func TestLoadConfig_EnvOverrides(t *testing.T) {
	t.Setenv("STOA_MCP_API_URL", "http://stoa:8080")
	t.Setenv("STOA_MCP_API_KEY", "test-key-123")
	t.Setenv("STOA_MCP_PORT", "8091")
	t.Setenv("STOA_MCP_BASE_URL", "https://mcp.example.com")

	cfg := LoadConfig()

	if cfg.APIURL != "http://stoa:8080" {
		t.Errorf("APIURL = %q, want %q", cfg.APIURL, "http://stoa:8080")
	}
	if cfg.APIKey != "test-key-123" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "test-key-123")
	}
	if cfg.Port != 8091 {
		t.Errorf("Port = %d, want %d", cfg.Port, 8091)
	}
	if cfg.BaseURL != "https://mcp.example.com" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "https://mcp.example.com")
	}
}

func TestLoadConfig_PortFallbackOnInvalidValue(t *testing.T) {
	t.Setenv("STOA_MCP_PORT", "not-a-number")
	t.Setenv("STOA_MCP_API_URL", "")
	t.Setenv("STOA_MCP_API_KEY", "")
	t.Setenv("STOA_MCP_BASE_URL", "")

	cfg := LoadConfig()

	// fmt.Sscanf fails silently, port stays at default.
	if cfg.Port != 8090 {
		t.Errorf("Port = %d, want %d (default on invalid input)", cfg.Port, 8090)
	}
}

func TestLoadConfig_BaseURLDerivedFromPort(t *testing.T) {
	t.Setenv("STOA_MCP_PORT", "9999")
	t.Setenv("STOA_MCP_BASE_URL", "")
	t.Setenv("STOA_MCP_API_URL", "")
	t.Setenv("STOA_MCP_API_KEY", "")

	cfg := LoadConfig()

	if cfg.BaseURL != "http://localhost:9999" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "http://localhost:9999")
	}
}

func TestEnvOrDefault(t *testing.T) {
	key := "STOA_TEST_ENV_OR_DEFAULT_KEY"

	os.Unsetenv(key)
	if got := envOrDefault(key, "fallback"); got != "fallback" {
		t.Errorf("envOrDefault unset = %q, want %q", got, "fallback")
	}

	t.Setenv(key, "override")
	if got := envOrDefault(key, "fallback"); got != "override" {
		t.Errorf("envOrDefault set = %q, want %q", got, "override")
	}

	t.Setenv(key, "")
	if got := envOrDefault(key, "fallback"); got != "fallback" {
		t.Errorf("envOrDefault empty = %q, want %q", got, "fallback")
	}
}

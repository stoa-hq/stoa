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

// --- LoadStoreConfig tests ---

func TestLoadStoreConfig_Defaults(t *testing.T) {
	for _, key := range []string{
		"STOA_STORE_MCP_API_URL", "STOA_STORE_MCP_API_KEY", "STOA_STORE_MCP_PORT", "STOA_STORE_MCP_BASE_URL",
		"STOA_MCP_API_URL", "STOA_MCP_API_KEY", "STOA_MCP_PORT", "STOA_MCP_BASE_URL",
	} {
		t.Setenv(key, "")
	}

	cfg := LoadStoreConfig()

	if cfg.APIURL != "http://localhost:8080" {
		t.Errorf("APIURL = %q, want %q", cfg.APIURL, "http://localhost:8080")
	}
	if cfg.APIKey != "" {
		t.Errorf("APIKey = %q, want empty", cfg.APIKey)
	}
	if cfg.Port != 8091 {
		t.Errorf("Port = %d, want %d", cfg.Port, 8091)
	}
	if cfg.BaseURL != "http://localhost:8091" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "http://localhost:8091")
	}
}

func TestLoadStoreConfig_StoreEnvOverrides(t *testing.T) {
	for _, key := range []string{"STOA_MCP_API_URL", "STOA_MCP_API_KEY", "STOA_MCP_PORT", "STOA_MCP_BASE_URL"} {
		t.Setenv(key, "")
	}
	t.Setenv("STOA_STORE_MCP_API_URL", "http://store:9090")
	t.Setenv("STOA_STORE_MCP_API_KEY", "sk_test123")
	t.Setenv("STOA_STORE_MCP_PORT", "9091")
	t.Setenv("STOA_STORE_MCP_BASE_URL", "https://store-mcp.example.com")

	cfg := LoadStoreConfig()

	if cfg.APIURL != "http://store:9090" {
		t.Errorf("APIURL = %q, want %q", cfg.APIURL, "http://store:9090")
	}
	if cfg.APIKey != "sk_test123" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "sk_test123")
	}
	if cfg.Port != 9091 {
		t.Errorf("Port = %d, want %d", cfg.Port, 9091)
	}
	if cfg.BaseURL != "https://store-mcp.example.com" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "https://store-mcp.example.com")
	}
}

func TestLoadStoreConfig_FallsBackToGeneric(t *testing.T) {
	for _, key := range []string{
		"STOA_STORE_MCP_API_URL", "STOA_STORE_MCP_API_KEY", "STOA_STORE_MCP_PORT", "STOA_STORE_MCP_BASE_URL",
	} {
		t.Setenv(key, "")
	}
	t.Setenv("STOA_MCP_API_URL", "http://generic:8080")
	t.Setenv("STOA_MCP_API_KEY", "ck_generic")
	t.Setenv("STOA_MCP_PORT", "7777")

	cfg := LoadStoreConfig()

	if cfg.APIURL != "http://generic:8080" {
		t.Errorf("APIURL = %q, want %q (fallback)", cfg.APIURL, "http://generic:8080")
	}
	if cfg.APIKey != "ck_generic" {
		t.Errorf("APIKey = %q, want %q (fallback)", cfg.APIKey, "ck_generic")
	}
	if cfg.Port != 7777 {
		t.Errorf("Port = %d, want %d (fallback)", cfg.Port, 7777)
	}
}

// --- LoadAdminConfig tests ---

func TestLoadAdminConfig_Defaults(t *testing.T) {
	for _, key := range []string{
		"STOA_ADMIN_MCP_API_URL", "STOA_ADMIN_MCP_API_KEY", "STOA_ADMIN_MCP_PORT", "STOA_ADMIN_MCP_BASE_URL",
		"STOA_MCP_API_URL", "STOA_MCP_API_KEY", "STOA_MCP_PORT", "STOA_MCP_BASE_URL",
	} {
		t.Setenv(key, "")
	}

	cfg := LoadAdminConfig()

	if cfg.APIURL != "http://localhost:8080" {
		t.Errorf("APIURL = %q, want %q", cfg.APIURL, "http://localhost:8080")
	}
	if cfg.APIKey != "" {
		t.Errorf("APIKey = %q, want empty", cfg.APIKey)
	}
	if cfg.Port != 8092 {
		t.Errorf("Port = %d, want %d", cfg.Port, 8092)
	}
	if cfg.BaseURL != "http://localhost:8092" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "http://localhost:8092")
	}
}

func TestLoadAdminConfig_AdminEnvOverrides(t *testing.T) {
	for _, key := range []string{"STOA_MCP_API_URL", "STOA_MCP_API_KEY", "STOA_MCP_PORT", "STOA_MCP_BASE_URL"} {
		t.Setenv(key, "")
	}
	t.Setenv("STOA_ADMIN_MCP_API_URL", "http://admin:9090")
	t.Setenv("STOA_ADMIN_MCP_API_KEY", "ck_admin_test")
	t.Setenv("STOA_ADMIN_MCP_PORT", "9092")
	t.Setenv("STOA_ADMIN_MCP_BASE_URL", "https://admin-mcp.example.com")

	cfg := LoadAdminConfig()

	if cfg.APIURL != "http://admin:9090" {
		t.Errorf("APIURL = %q, want %q", cfg.APIURL, "http://admin:9090")
	}
	if cfg.APIKey != "ck_admin_test" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "ck_admin_test")
	}
	if cfg.Port != 9092 {
		t.Errorf("Port = %d, want %d", cfg.Port, 9092)
	}
	if cfg.BaseURL != "https://admin-mcp.example.com" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "https://admin-mcp.example.com")
	}
}

func TestLoadAdminConfig_FallsBackToGeneric(t *testing.T) {
	for _, key := range []string{
		"STOA_ADMIN_MCP_API_URL", "STOA_ADMIN_MCP_API_KEY", "STOA_ADMIN_MCP_PORT", "STOA_ADMIN_MCP_BASE_URL",
	} {
		t.Setenv(key, "")
	}
	t.Setenv("STOA_MCP_API_URL", "http://generic:8080")
	t.Setenv("STOA_MCP_API_KEY", "ck_generic")
	t.Setenv("STOA_MCP_PORT", "6666")

	cfg := LoadAdminConfig()

	if cfg.APIURL != "http://generic:8080" {
		t.Errorf("APIURL = %q, want %q (fallback)", cfg.APIURL, "http://generic:8080")
	}
	if cfg.APIKey != "ck_generic" {
		t.Errorf("APIKey = %q, want %q (fallback)", cfg.APIKey, "ck_generic")
	}
	if cfg.Port != 6666 {
		t.Errorf("Port = %d, want %d (fallback)", cfg.Port, 6666)
	}
}

// --- envWithFallback tests ---

func TestEnvWithFallback(t *testing.T) {
	primary := "STOA_TEST_EWF_PRIMARY"
	fallback := "STOA_TEST_EWF_FALLBACK"

	// Both unset → default.
	t.Setenv(primary, "")
	t.Setenv(fallback, "")
	if got := envWithFallback(primary, fallback, "default"); got != "default" {
		t.Errorf("envWithFallback both unset = %q, want %q", got, "default")
	}

	// Primary set → primary wins.
	t.Setenv(primary, "primary-val")
	t.Setenv(fallback, "fallback-val")
	if got := envWithFallback(primary, fallback, "default"); got != "primary-val" {
		t.Errorf("envWithFallback primary set = %q, want %q", got, "primary-val")
	}

	// Only fallback set → fallback wins.
	t.Setenv(primary, "")
	t.Setenv(fallback, "fallback-val")
	if got := envWithFallback(primary, fallback, "default"); got != "fallback-val" {
		t.Errorf("envWithFallback fallback set = %q, want %q", got, "fallback-val")
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

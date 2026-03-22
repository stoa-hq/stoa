package mcp

import (
	"fmt"
	"os"
)

type Config struct {
	APIURL  string
	APIKey  string
	BaseURL string // Public base URL for SSE endpoint advertisement
	Port    int
}

// LoadConfig loads MCP configuration from generic STOA_MCP_* env vars.
// Deprecated: Use LoadStoreConfig or LoadAdminConfig instead.
func LoadConfig() *Config {
	cfg := &Config{
		APIURL: envOrDefault("STOA_MCP_API_URL", "http://localhost:8080"),
		APIKey: os.Getenv("STOA_MCP_API_KEY"),
		Port:   8090,
	}

	if port := os.Getenv("STOA_MCP_PORT"); port != "" {
		_, _ = fmt.Sscanf(port, "%d", &cfg.Port)
	}

	cfg.BaseURL = envOrDefault("STOA_MCP_BASE_URL", fmt.Sprintf("http://localhost:%d", cfg.Port))

	return cfg
}

// LoadStoreConfig loads Store MCP configuration.
// Checks STOA_STORE_MCP_* first, falls back to STOA_MCP_*.
func LoadStoreConfig() *Config {
	cfg := &Config{
		APIURL: envWithFallback("STOA_STORE_MCP_API_URL", "STOA_MCP_API_URL", "http://localhost:8080"),
		APIKey: envWithFallback("STOA_STORE_MCP_API_KEY", "STOA_MCP_API_KEY", ""),
		Port:   8091,
	}

	if port := envWithFallback("STOA_STORE_MCP_PORT", "STOA_MCP_PORT", ""); port != "" {
		_, _ = fmt.Sscanf(port, "%d", &cfg.Port)
	}

	cfg.BaseURL = envOrDefault("STOA_STORE_MCP_BASE_URL", fmt.Sprintf("http://localhost:%d", cfg.Port))

	return cfg
}

// LoadAdminConfig loads Admin MCP configuration.
// Checks STOA_ADMIN_MCP_* first, falls back to STOA_MCP_*.
func LoadAdminConfig() *Config {
	cfg := &Config{
		APIURL: envWithFallback("STOA_ADMIN_MCP_API_URL", "STOA_MCP_API_URL", "http://localhost:8080"),
		APIKey: envWithFallback("STOA_ADMIN_MCP_API_KEY", "STOA_MCP_API_KEY", ""),
		Port:   8092,
	}

	if port := envWithFallback("STOA_ADMIN_MCP_PORT", "STOA_MCP_PORT", ""); port != "" {
		_, _ = fmt.Sscanf(port, "%d", &cfg.Port)
	}

	cfg.BaseURL = envOrDefault("STOA_ADMIN_MCP_BASE_URL", fmt.Sprintf("http://localhost:%d", cfg.Port))

	return cfg
}

func envWithFallback(primary, fallback, defaultVal string) string {
	if v := os.Getenv(primary); v != "" {
		return v
	}
	if v := os.Getenv(fallback); v != "" {
		return v
	}
	return defaultVal
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

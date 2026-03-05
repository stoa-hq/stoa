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

func LoadConfig() *Config {
	cfg := &Config{
		APIURL: envOrDefault("STOA_MCP_API_URL", "http://localhost:8080"),
		APIKey: os.Getenv("STOA_MCP_API_KEY"),
		Port:   8090,
	}

	if port := os.Getenv("STOA_MCP_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &cfg.Port)
	}

	cfg.BaseURL = envOrDefault("STOA_MCP_BASE_URL", fmt.Sprintf("http://localhost:%d", cfg.Port))

	return cfg
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

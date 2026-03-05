package mcp

import (
	"fmt"
	"os"
)

type Config struct {
	APIURL    string
	APIKey    string
	Transport string // "stdio" or "http"
	HTTPPort  int
}

func LoadConfig() *Config {
	cfg := &Config{
		APIURL:    envOrDefault("STOA_MCP_API_URL", "http://localhost:8080"),
		APIKey:    os.Getenv("STOA_MCP_API_KEY"),
		Transport: envOrDefault("STOA_MCP_TRANSPORT", "stdio"),
		HTTPPort:  8090,
	}

	if port := os.Getenv("STOA_MCP_HTTP_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &cfg.HTTPPort)
	}

	return cfg
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

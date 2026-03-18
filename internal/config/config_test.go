package config

import (
	"testing"
)

func TestDefaults_EndpointRateLimits(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	tests := []struct {
		name string
		got  int
		want int
	}{
		{"global requests_per_minute", cfg.Security.RateLimit.RequestsPerMinute, 300},
		{"global burst", cfg.Security.RateLimit.Burst, 50},
		{"login requests_per_minute", cfg.Security.RateLimit.Login.RequestsPerMinute, 10},
		{"register requests_per_minute", cfg.Security.RateLimit.Register.RequestsPerMinute, 5},
		{"checkout requests_per_minute", cfg.Security.RateLimit.Checkout.RequestsPerMinute, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %d, want %d", tt.got, tt.want)
			}
		})
	}
}

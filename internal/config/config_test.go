package config

import (
	"strings"
	"testing"
)

func TestValidate_PaymentEncryptionKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr string
	}{
		{"empty key", "", "required"},
		{"valid 32-byte raw key", "abcdefghijklmnopqrstuvwxyz012345", ""},
		{"valid 64 hex chars", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", ""},
		{"too short 16 chars", "0123456789abcdef", "must be exactly 32 bytes"},
		{"too long 48 chars", "0123456789abcdef0123456789abcdef0123456789abcdef", "must be exactly 32 bytes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Payment: PaymentConfig{EncryptionKey: tt.key},
			}
			err := cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q should contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

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

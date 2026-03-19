package server

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/internal/config"
)

func newTestServer(origins []string) (*Server, *bytes.Buffer) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	cfg := &config.Config{}
	cfg.Server.CORS.AllowedOrigins = origins
	return &Server{cfg: cfg, logger: logger}, &buf
}

func TestValidateCORS_WildcardReturnsError(t *testing.T) {
	s, _ := newTestServer([]string{"*"})
	err := s.validateCORS()
	if err == nil {
		t.Fatal("expected error for wildcard origin with credentials, got nil")
	}
	if !strings.Contains(err.Error(), "wildcard") {
		t.Errorf("error should mention wildcard, got: %s", err.Error())
	}
}

func TestValidateCORS_WildcardAmongOthersReturnsError(t *testing.T) {
	s, _ := newTestServer([]string{"https://example.com", "*"})
	err := s.validateCORS()
	if err == nil {
		t.Fatal("expected error when wildcard is among other origins")
	}
}

func TestValidateCORS_ExplicitOriginsOK(t *testing.T) {
	s, _ := newTestServer([]string{"https://example.com", "https://admin.example.com"})
	err := s.validateCORS()
	if err != nil {
		t.Fatalf("expected no error for explicit origins, got: %v", err)
	}
}

func TestValidateCORS_ManyOriginsWarns(t *testing.T) {
	origins := []string{
		"https://a.com",
		"https://b.com",
		"https://c.com",
		"https://d.com",
	}
	s, buf := newTestServer(origins)
	err := s.validateCORS()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "large number of allowed origins") {
		t.Error("expected warning log for many origins")
	}
}

func TestValidateCORS_EmptyOriginsOK(t *testing.T) {
	s, _ := newTestServer(nil)
	err := s.validateCORS()
	if err != nil {
		t.Fatalf("expected no error for empty origins, got: %v", err)
	}
}

package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/internal/config"
)

func TestSecurityHeaders(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	cfg := &config.Config{}
	cfg.Server.CORS.AllowedOrigins = []string{"https://example.com"}
	cfg.Security.RateLimit.RequestsPerMinute = 100
	cfg.Security.CSRF.Secure = false

	s := &Server{
		cfg:    cfg,
		logger: logger,
		router: chi.NewRouter(),
	}
	s.setupMiddleware()
	s.router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	expected := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":       "DENY",
		"X-Xss-Protection":      "1; mode=block",
		"Referrer-Policy":       "strict-origin-when-cross-origin",
		"Content-Security-Policy": "default-src 'self'",
		"Permissions-Policy":    "camera=(), microphone=(), geolocation=()",
	}

	for header, want := range expected {
		got := rec.Header().Get(header)
		if got != want {
			t.Errorf("header %s = %q, want %q", header, got, want)
		}
	}
}

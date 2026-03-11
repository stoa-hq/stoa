package settings

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/internal/config"
)

func newTestHandler() *handler {
	cfg := &config.Config{
		I18n: config.I18nConfig{
			DefaultLocale:    "de-DE",
			AvailableLocales: []string{"de-DE", "en-US"},
		},
	}
	return NewHandler(cfg, zerolog.Nop())
}

func TestGetConfig(t *testing.T) {
	h := newTestHandler()

	r := chi.NewRouter()
	r.Get("/config", h.getConfig)

	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", contentType)
	}

	var resp struct {
		Data ConfigResponse `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Data.DefaultLocale != "de-DE" {
		t.Errorf("expected default_locale %q, got %q", "de-DE", resp.Data.DefaultLocale)
	}
	if len(resp.Data.AvailableLocales) != 2 {
		t.Fatalf("expected 2 available_locales, got %d", len(resp.Data.AvailableLocales))
	}
	if resp.Data.AvailableLocales[0] != "de-DE" {
		t.Errorf("expected first locale %q, got %q", "de-DE", resp.Data.AvailableLocales[0])
	}
	if resp.Data.AvailableLocales[1] != "en-US" {
		t.Errorf("expected second locale %q, got %q", "en-US", resp.Data.AvailableLocales[1])
	}
}

func TestRegisterStoreRoutes(t *testing.T) {
	h := newTestHandler()

	r := chi.NewRouter()
	h.RegisterStoreRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 from store route, got %d", rec.Code)
	}

	var resp struct {
		Data ConfigResponse `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Data.DefaultLocale != "de-DE" {
		t.Errorf("expected default_locale %q, got %q", "de-DE", resp.Data.DefaultLocale)
	}
}

func TestRegisterAdminRoutes(t *testing.T) {
	h := newTestHandler()

	r := chi.NewRouter()
	h.RegisterAdminRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 from admin route, got %d", rec.Code)
	}

	var resp struct {
		Data ConfigResponse `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Data.DefaultLocale != "de-DE" {
		t.Errorf("expected default_locale %q, got %q", "de-DE", resp.Data.DefaultLocale)
	}
}

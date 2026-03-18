package settings

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/internal/config"
)

func newTestHandler(repo ...Repository) *handler {
	var r Repository
	if len(repo) > 0 {
		r = repo[0]
	} else {
		r = &mockRepo{}
	}
	svc := NewService(r, zerolog.Nop())
	cfg := &config.Config{
		I18n: config.I18nConfig{
			DefaultLocale:    "de-DE",
			AvailableLocales: []string{"de-DE", "en-US"},
		},
	}
	return NewHandler(svc, cfg, validator.New(), zerolog.Nop())
}

// ---------------------------------------------------------------------------
// GET /config (existing functionality)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// GET /settings (store)
// ---------------------------------------------------------------------------

func TestHandler_GetSettings_Store(t *testing.T) {
	repo := &mockRepo{
		get: func(_ context.Context) (*StoreSettings, error) {
			return &StoreSettings{StoreName: "Test Shop", Currency: "EUR", Timezone: "UTC"}, nil
		},
	}
	h := newTestHandler(repo)

	r := chi.NewRouter()
	h.RegisterStoreRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Data StoreSettings `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Data.StoreName != "Test Shop" {
		t.Errorf("expected 'Test Shop', got %q", resp.Data.StoreName)
	}
}

func TestHandler_GetSettings_DefaultFallback(t *testing.T) {
	h := newTestHandler() // mockRepo returns ErrNotFound → defaults

	r := chi.NewRouter()
	h.RegisterStoreRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Data StoreSettings `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Data.StoreName != "Stoa" {
		t.Errorf("expected default 'Stoa', got %q", resp.Data.StoreName)
	}
}

// ---------------------------------------------------------------------------
// PUT /settings (admin)
// ---------------------------------------------------------------------------

func TestHandler_UpdateSettings_Success(t *testing.T) {
	repo := &mockRepo{
		upsert: func(_ context.Context, s *StoreSettings) (*StoreSettings, error) {
			return s, nil
		},
	}
	h := newTestHandler(repo)

	r := chi.NewRouter()
	h.RegisterAdminRoutes(r)

	body := UpdateSettingsRequest{
		StoreName: "New Name",
		Currency:  "USD",
		Timezone:  "America/New_York",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/settings", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data StoreSettings `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Data.StoreName != "New Name" {
		t.Errorf("expected 'New Name', got %q", resp.Data.StoreName)
	}
}

func TestHandler_UpdateSettings_InvalidJSON(t *testing.T) {
	h := newTestHandler()

	r := chi.NewRouter()
	h.RegisterAdminRoutes(r)

	req := httptest.NewRequest(http.MethodPut, "/settings", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandler_UpdateSettings_ValidationError(t *testing.T) {
	h := newTestHandler()

	r := chi.NewRouter()
	h.RegisterAdminRoutes(r)

	body := UpdateSettingsRequest{
		StoreName: "",
		Currency:  "EUR",
		Timezone:  "UTC",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/settings", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d; body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_UpdateSettings_InvalidCurrency(t *testing.T) {
	h := newTestHandler()

	r := chi.NewRouter()
	h.RegisterAdminRoutes(r)

	body := UpdateSettingsRequest{
		StoreName: "Shop",
		Currency:  "EURO",
		Timezone:  "UTC",
	}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/settings", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d; body: %s", w.Code, w.Body.String())
	}
}

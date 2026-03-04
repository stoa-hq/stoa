package product

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/epoxx-arch/stoa/pkg/sdk"
)

// newTestHandler wires a Handler with the given mock repository.
func newTestHandler(repo ProductRepository) *Handler {
	svc := NewService(repo, sdk.NewHookRegistry(), zerolog.Nop())
	return NewHandler(svc, validator.New(), zerolog.Nop())
}

// withChiParam injects a chi URL parameter into the request context.
func withChiParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// ---------------------------------------------------------------------------
// GET /products/{id}  (admin)
// ---------------------------------------------------------------------------

func TestHandler_AdminGetByID_NotFound(t *testing.T) {
	id := uuid.New()
	req := withChiParam(
		httptest.NewRequest(http.MethodGet, "/products/"+id.String(), nil),
		"id", id.String(),
	)
	w := httptest.NewRecorder()

	newTestHandler(&mockRepo{}).adminGetByID(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", w.Code)
	}
}

func TestHandler_AdminGetByID_InvalidUUID(t *testing.T) {
	req := withChiParam(
		httptest.NewRequest(http.MethodGet, "/products/bad-id", nil),
		"id", "bad-id",
	)
	w := httptest.NewRecorder()

	newTestHandler(&mockRepo{}).adminGetByID(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

func TestHandler_AdminGetByID_Success(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		findByID: func(_ context.Context, _ uuid.UUID) (*Product, error) {
			return &Product{ID: id, SKU: "FOUND", Currency: "EUR"}, nil
		},
	}

	req := withChiParam(
		httptest.NewRequest(http.MethodGet, "/products/"+id.String(), nil),
		"id", id.String(),
	)
	w := httptest.NewRecorder()

	newTestHandler(repo).adminGetByID(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", w.Code)
	}
	var resp apiResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data == nil {
		t.Error("expected data in response")
	}
}

// ---------------------------------------------------------------------------
// POST /products  (admin create)
// ---------------------------------------------------------------------------

func TestHandler_AdminCreate_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBufferString("not-json"))
	w := httptest.NewRecorder()

	newTestHandler(&mockRepo{}).adminCreate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

func TestHandler_AdminCreate_ValidationError(t *testing.T) {
	// Missing required "currency" and "translations".
	body, _ := json.Marshal(map[string]interface{}{"sku": "TEST"})
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	newTestHandler(&mockRepo{}).adminCreate(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status: got %d, want 422; body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_AdminCreate_Success(t *testing.T) {
	created := false
	repo := &mockRepo{
		create: func(_ context.Context, _ *Product) error {
			created = true
			return nil
		},
	}

	body, _ := json.Marshal(CreateProductRequest{
		SKU:      "NEW-001",
		Currency: "EUR",
		Translations: []TranslationInput{
			{Locale: "de-DE", Name: "Neues Produkt", Slug: "neues-produkt"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	newTestHandler(repo).adminCreate(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status: got %d, want 201; body: %s", w.Code, w.Body.String())
	}
	if !created {
		t.Error("expected repo.Create to be called")
	}
}

// ---------------------------------------------------------------------------
// DELETE /products/{id}  (admin)
// ---------------------------------------------------------------------------

func TestHandler_AdminDelete_NotFound(t *testing.T) {
	id := uuid.New()
	req := withChiParam(
		httptest.NewRequest(http.MethodDelete, "/products/"+id.String(), nil),
		"id", id.String(),
	)
	w := httptest.NewRecorder()

	// mockRepo.FindByID returns ErrNotFound → service.Delete fails.
	newTestHandler(&mockRepo{}).adminDelete(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", w.Code)
	}
}

func TestHandler_AdminDelete_Success(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		findByID: func(_ context.Context, _ uuid.UUID) (*Product, error) {
			return &Product{ID: id}, nil
		},
		delete: func(_ context.Context, _ uuid.UUID) error {
			return nil
		},
	}

	req := withChiParam(
		httptest.NewRequest(http.MethodDelete, "/products/"+id.String(), nil),
		"id", id.String(),
	)
	w := httptest.NewRecorder()

	newTestHandler(repo).adminDelete(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status: got %d, want 204", w.Code)
	}
}

// ---------------------------------------------------------------------------
// GET /products/{slug}  (store)
// ---------------------------------------------------------------------------

func TestHandler_StoreGetBySlug_NotFound(t *testing.T) {
	req := withChiParam(
		httptest.NewRequest(http.MethodGet, "/products/no-such-slug", nil),
		"slug", "no-such-slug",
	)
	w := httptest.NewRecorder()

	newTestHandler(&mockRepo{}).storeGetBySlug(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", w.Code)
	}
}

func TestHandler_StoreGetBySlug_Success(t *testing.T) {
	repo := &mockRepo{
		findBySlug: func(_ context.Context, slug, _ string) (*Product, error) {
			return &Product{ID: uuid.New(), SKU: "SLUG-PRODUCT"}, nil
		},
	}

	req := withChiParam(
		httptest.NewRequest(http.MethodGet, "/products/my-product", nil),
		"slug", "my-product",
	)
	w := httptest.NewRecorder()

	newTestHandler(repo).storeGetBySlug(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200; body: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// parseLocale helper
// ---------------------------------------------------------------------------

func TestParseLocale_AcceptLanguageHeader(t *testing.T) {
	tests := []struct {
		header string
		want   string
	}{
		{"de-DE,de;q=0.9,en;q=0.8", "de-DE"},
		{"en-US", "en-US"},
		{"fr;q=0.7", "fr"},
		{"", "en"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if tt.header != "" {
			req.Header.Set("Accept-Language", tt.header)
		}
		got := parseLocale(req)
		if got != tt.want {
			t.Errorf("header=%q: got %q, want %q", tt.header, got, tt.want)
		}
	}
}

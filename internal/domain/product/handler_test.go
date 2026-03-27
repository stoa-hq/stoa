package product

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/pkg/sdk"
)

// newTestHandler wires a Handler with the given mock repository.
func newTestHandler(repo ProductRepository) *Handler {
	noopURL := func(s string) string { return "/uploads/" + s }
	noopTax := TaxRateFn(func(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil })
	svc := NewService(repo, sdk.NewHookRegistry(), zerolog.Nop(), noopURL, noopTax)
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

func TestHandler_AdminList_ErrorDoesNotLeakInternalDetails(t *testing.T) {
	internalMsg := "pq: relation \"products\" does not exist"
	repo := &mockRepo{
		findAll: func(_ context.Context, _ ProductFilter) ([]Product, int, error) {
			return nil, 0, errors.New(internalMsg)
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	w := httptest.NewRecorder()

	newTestHandler(repo).adminList(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want 500", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "an unexpected error occurred") {
		t.Errorf("response should contain generic error message, got: %s", body)
	}
	if strings.Contains(body, internalMsg) {
		t.Errorf("response must NOT contain internal error detail %q, got: %s", internalMsg, body)
	}
}

// ---------------------------------------------------------------------------
// parseLocale helper
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// POST /property-groups  (admin create property group)
// ---------------------------------------------------------------------------

func TestHandler_AdminCreatePropertyGroup_Success(t *testing.T) {
	repo := &mockRepo{}
	body, _ := json.Marshal(CreatePropertyGroupRequest{
		Identifier:   "color",
		Translations: []PropertyGroupTranslationInput{{Locale: "en-US", Name: "Color"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/property-groups", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	newTestHandler(repo).adminCreatePropertyGroup(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status: got %d, want 201; body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_AdminCreatePropertyGroup_MissingIdentifier(t *testing.T) {
	body, _ := json.Marshal(map[string]any{
		"translations": []map[string]string{{"locale": "en-US", "name": "Color"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/property-groups", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	newTestHandler(&mockRepo{}).adminCreatePropertyGroup(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status: got %d, want 422; body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_AdminCreatePropertyGroup_DuplicateIdentifier(t *testing.T) {
	repo := &mockRepo{
		createPropGroup: func(_ *PropertyGroup) error {
			return ErrDuplicateIdentifier
		},
	}
	body, _ := json.Marshal(CreatePropertyGroupRequest{
		Identifier:   "color",
		Translations: []PropertyGroupTranslationInput{{Locale: "en-US", Name: "Color"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/property-groups", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	newTestHandler(repo).adminCreatePropertyGroup(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status: got %d, want 409; body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_AdminCreatePropertyGroup_InvalidIdentifier(t *testing.T) {
	body, _ := json.Marshal(CreatePropertyGroupRequest{
		Identifier:   "INVALID ID!",
		Translations: []PropertyGroupTranslationInput{{Locale: "en-US", Name: "Color"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/property-groups", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	newTestHandler(&mockRepo{}).adminCreatePropertyGroup(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status: got %d, want 422; body: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// PUT /property-groups/{id}  (admin update property group)
// ---------------------------------------------------------------------------

func TestHandler_AdminUpdatePropertyGroup_DuplicateIdentifier(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		updatePropGroup: func(_ *PropertyGroup) error {
			return ErrDuplicateIdentifier
		},
	}
	body, _ := json.Marshal(CreatePropertyGroupRequest{
		Identifier:   "color",
		Translations: []PropertyGroupTranslationInput{{Locale: "en-US", Name: "Color"}},
	})
	req := withChiParam(
		httptest.NewRequest(http.MethodPut, "/property-groups/"+id.String(), bytes.NewReader(body)),
		"id", id.String(),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	newTestHandler(repo).adminUpdatePropertyGroup(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status: got %d, want 409; body: %s", w.Code, w.Body.String())
	}
}

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

// ---------------------------------------------------------------------------
// GET /attributes  (admin list attributes)
// ---------------------------------------------------------------------------

func TestHandler_AdminListAttributes_Success(t *testing.T) {
	repo := &mockRepo{
		findAllAttributes: func() ([]Attribute, error) {
			return []Attribute{
				{ID: uuid.New(), Identifier: "material", Type: "text"},
				{ID: uuid.New(), Identifier: "weight-kg", Type: "number"},
			}, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/attributes", nil)
	w := httptest.NewRecorder()

	newTestHandler(repo).adminListAttributes(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200; body: %s", w.Code, w.Body.String())
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
// POST /attributes  (admin create attribute)
// ---------------------------------------------------------------------------

func TestHandler_AdminCreateAttribute_Success(t *testing.T) {
	body, _ := json.Marshal(CreateAttributeRequest{
		Identifier:   "material",
		Type:         "text",
		Translations: []AttributeTranslationInput{{Locale: "en-US", Name: "Material"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/attributes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	newTestHandler(&mockRepo{}).adminCreateAttribute(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status: got %d, want 201; body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_AdminCreateAttribute_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/attributes", bytes.NewBufferString("not-json"))
	w := httptest.NewRecorder()

	newTestHandler(&mockRepo{}).adminCreateAttribute(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

func TestHandler_AdminCreateAttribute_MissingTranslations(t *testing.T) {
	body, _ := json.Marshal(map[string]any{
		"identifier": "material",
		"type":       "text",
	})
	req := httptest.NewRequest(http.MethodPost, "/attributes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	newTestHandler(&mockRepo{}).adminCreateAttribute(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status: got %d, want 422; body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_AdminCreateAttribute_InvalidIdentifier(t *testing.T) {
	body, _ := json.Marshal(CreateAttributeRequest{
		Identifier:   "INVALID ID!",
		Type:         "text",
		Translations: []AttributeTranslationInput{{Locale: "en-US", Name: "Material"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/attributes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	newTestHandler(&mockRepo{}).adminCreateAttribute(w, req)

	// validator rejects the identifier format at the handler level (min=1,max=255 passes,
	// but service rejects the format pattern — the handler maps ErrInvalidIdentifier to 422).
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status: got %d, want 422; body: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GET /attributes/{id}  (admin get attribute)
// ---------------------------------------------------------------------------

func TestHandler_AdminGetAttribute_NotFound(t *testing.T) {
	id := uuid.New()
	req := withChiParam(
		httptest.NewRequest(http.MethodGet, "/attributes/"+id.String(), nil),
		"id", id.String(),
	)
	w := httptest.NewRecorder()

	// mockRepo.FindAttributeByID returns ErrNotFound by default.
	newTestHandler(&mockRepo{}).adminGetAttribute(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", w.Code)
	}
}

func TestHandler_AdminGetAttribute_Success(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		findAttributeByID: func(got uuid.UUID) (*Attribute, error) {
			return &Attribute{ID: got, Identifier: "material", Type: "text"}, nil
		},
	}

	req := withChiParam(
		httptest.NewRequest(http.MethodGet, "/attributes/"+id.String(), nil),
		"id", id.String(),
	)
	w := httptest.NewRecorder()

	newTestHandler(repo).adminGetAttribute(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var resp apiResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data == nil {
		t.Error("expected data in response")
	}
}

func TestHandler_AdminGetAttribute_InvalidUUID(t *testing.T) {
	req := withChiParam(
		httptest.NewRequest(http.MethodGet, "/attributes/bad-id", nil),
		"id", "bad-id",
	)
	w := httptest.NewRecorder()

	newTestHandler(&mockRepo{}).adminGetAttribute(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

// ---------------------------------------------------------------------------
// DELETE /attributes/{id}  (admin delete attribute)
// ---------------------------------------------------------------------------

func TestHandler_AdminDeleteAttribute_Success(t *testing.T) {
	id := uuid.New()
	deleted := false
	repo := &mockRepo{
		deleteAttribute: func(_ uuid.UUID) error {
			deleted = true
			return nil
		},
	}

	req := withChiParam(
		httptest.NewRequest(http.MethodDelete, "/attributes/"+id.String(), nil),
		"id", id.String(),
	)
	w := httptest.NewRecorder()

	newTestHandler(repo).adminDeleteAttribute(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status: got %d, want 204; body: %s", w.Code, w.Body.String())
	}
	if !deleted {
		t.Error("expected repo.DeleteAttribute to be called")
	}
}

func TestHandler_AdminDeleteAttribute_InvalidUUID(t *testing.T) {
	req := withChiParam(
		httptest.NewRequest(http.MethodDelete, "/attributes/bad-id", nil),
		"id", "bad-id",
	)
	w := httptest.NewRecorder()

	newTestHandler(&mockRepo{}).adminDeleteAttribute(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

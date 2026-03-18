package warehouse

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

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newTestHandler(repo WarehouseRepository) *Handler {
	svc := NewService(repo, sdk.NewHookRegistry(), zerolog.Nop())
	return NewHandler(svc, validator.New(), zerolog.Nop())
}

func withChiParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestHandler_List_Empty(t *testing.T) {
	h := newTestHandler(&mockWarehouseRepo{})
	req := httptest.NewRequest(http.MethodGet, "/warehouses", nil)
	w := httptest.NewRecorder()
	h.list(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", w.Code)
	}

	var resp apiResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Meta == nil {
		t.Fatal("expected meta in response")
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestHandler_Create_Success(t *testing.T) {
	h := newTestHandler(&mockWarehouseRepo{})
	body := `{"name":"Test WH","code":"TEST","active":true,"priority":0}`
	req := httptest.NewRequest(http.MethodPost, "/warehouses", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.create(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status: got %d, want 201. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_Create_InvalidJSON(t *testing.T) {
	h := newTestHandler(&mockWarehouseRepo{})
	req := httptest.NewRequest(http.MethodPost, "/warehouses", bytes.NewBufferString("{invalid"))
	w := httptest.NewRecorder()
	h.create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

func TestHandler_Create_ValidationError(t *testing.T) {
	h := newTestHandler(&mockWarehouseRepo{})
	// Missing required fields.
	body := `{"active":true}`
	req := httptest.NewRequest(http.MethodPost, "/warehouses", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.create(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status: got %d, want 422. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_Create_DuplicateCode(t *testing.T) {
	repo := &mockWarehouseRepo{
		create: func(_ context.Context, _ *Warehouse) error {
			return ErrDuplicateCode
		},
	}
	h := newTestHandler(repo)
	body := `{"name":"WH","code":"DUP","active":true}`
	req := httptest.NewRequest(http.MethodPost, "/warehouses", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.create(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status: got %d, want 409. Body: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestHandler_GetByID_InvalidUUID(t *testing.T) {
	h := newTestHandler(&mockWarehouseRepo{})
	req := withChiParam(
		httptest.NewRequest(http.MethodGet, "/warehouses/not-a-uuid", nil),
		"id", "not-a-uuid",
	)
	w := httptest.NewRecorder()
	h.getByID(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

func TestHandler_GetByID_NotFound(t *testing.T) {
	h := newTestHandler(&mockWarehouseRepo{})
	id := uuid.New()
	req := withChiParam(
		httptest.NewRequest(http.MethodGet, "/warehouses/"+id.String(), nil),
		"id", id.String(),
	)
	w := httptest.NewRecorder()
	h.getByID(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", w.Code)
	}
}

func TestHandler_GetByID_Success(t *testing.T) {
	id := uuid.New()
	repo := &mockWarehouseRepo{
		findByID: func(_ context.Context, _ uuid.UUID) (*Warehouse, error) {
			return &Warehouse{ID: id, Name: "WH1", Code: "WH1"}, nil
		},
	}
	h := newTestHandler(repo)
	req := withChiParam(
		httptest.NewRequest(http.MethodGet, "/warehouses/"+id.String(), nil),
		"id", id.String(),
	)
	w := httptest.NewRecorder()
	h.getByID(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestHandler_Update_Success(t *testing.T) {
	h := newTestHandler(&mockWarehouseRepo{})
	id := uuid.New()
	body := `{"name":"Updated","code":"UPD","active":true,"priority":1}`
	req := withChiParam(
		httptest.NewRequest(http.MethodPut, "/warehouses/"+id.String(), bytes.NewBufferString(body)),
		"id", id.String(),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200. Body: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestHandler_Delete_Success(t *testing.T) {
	h := newTestHandler(&mockWarehouseRepo{})
	id := uuid.New()
	req := withChiParam(
		httptest.NewRequest(http.MethodDelete, "/warehouses/"+id.String(), nil),
		"id", id.String(),
	)
	w := httptest.NewRecorder()
	h.delete(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status: got %d, want 204", w.Code)
	}
}

func TestHandler_Delete_NotFound(t *testing.T) {
	repo := &mockWarehouseRepo{
		delete: func(_ context.Context, _ uuid.UUID) error {
			return ErrNotFound
		},
	}
	h := newTestHandler(repo)
	id := uuid.New()
	req := withChiParam(
		httptest.NewRequest(http.MethodDelete, "/warehouses/"+id.String(), nil),
		"id", id.String(),
	)
	w := httptest.NewRecorder()
	h.delete(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", w.Code)
	}
}

// ---------------------------------------------------------------------------
// SetStock
// ---------------------------------------------------------------------------

func TestHandler_SetStock_Success(t *testing.T) {
	h := newTestHandler(&mockWarehouseRepo{})
	whID := uuid.New()
	pID := uuid.New()
	body, _ := json.Marshal(SetStockRequest{
		Items: []SetStockItem{
			{ProductID: pID, Quantity: 100, Reference: "initial"},
		},
	})
	req := withChiParam(
		httptest.NewRequest(http.MethodPut, "/warehouses/"+whID.String()+"/stock", bytes.NewBuffer(body)),
		"id", whID.String(),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.setStock(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200. Body: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GetStockByProduct
// ---------------------------------------------------------------------------

func TestHandler_GetStockByProduct_Success(t *testing.T) {
	pID := uuid.New()
	repo := &mockWarehouseRepo{
		getStockByProduct: func(_ context.Context, _ uuid.UUID) ([]WarehouseStock, error) {
			return []WarehouseStock{
				{ID: uuid.New(), ProductID: pID, Quantity: 50, WarehouseName: "WH1", ProductSKU: "SKU-001", ProductName: "Test Product"},
			}, nil
		},
	}
	h := newTestHandler(repo)
	req := withChiParam(
		httptest.NewRequest(http.MethodGet, "/products/"+pID.String()+"/stock", nil),
		"productID", pID.String(),
	)
	w := httptest.NewRecorder()
	h.getStockByProduct(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", w.Code)
	}

	// Verify SKU fields are present in response.
	var resp struct {
		Data []struct {
			ProductSKU  string `json:"product_sku"`
			ProductName string `json:"product_name"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 stock entry, got %d", len(resp.Data))
	}
	if resp.Data[0].ProductSKU != "SKU-001" {
		t.Errorf("product_sku: got %q, want %q", resp.Data[0].ProductSKU, "SKU-001")
	}
	if resp.Data[0].ProductName != "Test Product" {
		t.Errorf("product_name: got %q, want %q", resp.Data[0].ProductName, "Test Product")
	}
}

func TestHandler_GetStockByWarehouse_ContainsSKU(t *testing.T) {
	whID := uuid.New()
	repo := &mockWarehouseRepo{
		getStockByWarehouse: func(_ context.Context, _ uuid.UUID) ([]WarehouseStock, error) {
			return []WarehouseStock{
				{
					ID: uuid.New(), WarehouseID: whID, ProductID: uuid.New(),
					Quantity: 10, WarehouseName: "WH1", WarehouseCode: "WH1",
					ProductSKU: "PROD-A", ProductName: "Product A", VariantSKU: "VAR-1",
				},
			}, nil
		},
	}
	h := newTestHandler(repo)
	req := withChiParam(
		httptest.NewRequest(http.MethodGet, "/warehouses/"+whID.String()+"/stock", nil),
		"id", whID.String(),
	)
	w := httptest.NewRecorder()
	h.getStockByWarehouse(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", w.Code)
	}

	var resp struct {
		Data []struct {
			ProductSKU  string `json:"product_sku"`
			ProductName string `json:"product_name"`
			VariantSKU  string `json:"variant_sku"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 stock entry, got %d", len(resp.Data))
	}
	if resp.Data[0].ProductSKU != "PROD-A" {
		t.Errorf("product_sku: got %q, want %q", resp.Data[0].ProductSKU, "PROD-A")
	}
	if resp.Data[0].VariantSKU != "VAR-1" {
		t.Errorf("variant_sku: got %q, want %q", resp.Data[0].VariantSKU, "VAR-1")
	}
}

// ---------------------------------------------------------------------------
// RemoveStock
// ---------------------------------------------------------------------------

func TestHandler_RemoveStock_Success(t *testing.T) {
	h := newTestHandler(&mockWarehouseRepo{})
	whID := uuid.New()
	stockID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/warehouses/"+whID.String()+"/stock/"+stockID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", whID.String())
	rctx.URLParams.Add("stockID", stockID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	h.removeStock(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status: got %d, want 204. Body: %s", w.Code, w.Body.String())
	}
}

func TestHandler_RemoveStock_NotFound(t *testing.T) {
	repo := &mockWarehouseRepo{
		removeStock: func(_ context.Context, _ uuid.UUID) error {
			return ErrNotFound
		},
	}
	h := newTestHandler(repo)
	whID := uuid.New()
	stockID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/warehouses/"+whID.String()+"/stock/"+stockID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", whID.String())
	rctx.URLParams.Add("stockID", stockID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	h.removeStock(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", w.Code)
	}
}

// ---------------------------------------------------------------------------
// List — error information disclosure
// ---------------------------------------------------------------------------

func TestHandler_List_ServiceError_NoInfoDisclosure(t *testing.T) {
	repo := &mockWarehouseRepo{
		findAll: func(_ context.Context, _ WarehouseFilter) ([]Warehouse, int, error) {
			return nil, 0, errors.New("pq: relation \"warehouses\" does not exist")
		},
	}
	h := newTestHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/warehouses", nil)
	w := httptest.NewRecorder()
	h.list(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want 500", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "an unexpected error occurred") {
		t.Errorf("expected generic error message in response body, got: %s", body)
	}
	if strings.Contains(body, "warehouses") {
		t.Errorf("response body must not contain internal error details, got: %s", body)
	}
}

func TestHandler_RemoveStock_InvalidUUID(t *testing.T) {
	h := newTestHandler(&mockWarehouseRepo{})
	whID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/warehouses/"+whID.String()+"/stock/not-a-uuid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", whID.String())
	rctx.URLParams.Add("stockID", "not-a-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	h.removeStock(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

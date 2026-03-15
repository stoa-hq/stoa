package payment

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ---------------------------------------------------------------------------
// Mock services for handler tests
// ---------------------------------------------------------------------------

type mockMethodSvc struct {
	list    func(ctx context.Context, f PaymentMethodFilter) ([]PaymentMethod, int, error)
	getByID func(ctx context.Context, id uuid.UUID) (*PaymentMethod, error)
	create  func(ctx context.Context, m *PaymentMethod) error
	update  func(ctx context.Context, m *PaymentMethod) error
	delete  func(ctx context.Context, id uuid.UUID) error
}

func (m *mockMethodSvc) List(ctx context.Context, f PaymentMethodFilter) ([]PaymentMethod, int, error) {
	if m.list != nil {
		return m.list(ctx, f)
	}
	return nil, 0, nil
}
func (m *mockMethodSvc) GetByID(ctx context.Context, id uuid.UUID) (*PaymentMethod, error) {
	if m.getByID != nil {
		return m.getByID(ctx, id)
	}
	return nil, ErrMethodNotFound
}
func (m *mockMethodSvc) Create(ctx context.Context, pm *PaymentMethod) error {
	if m.create != nil {
		return m.create(ctx, pm)
	}
	return nil
}
func (m *mockMethodSvc) Update(ctx context.Context, pm *PaymentMethod) error {
	if m.update != nil {
		return m.update(ctx, pm)
	}
	return nil
}
func (m *mockMethodSvc) Delete(ctx context.Context, id uuid.UUID) error {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return nil
}

type mockTxSvc struct {
	createTransaction      func(ctx context.Context, t *PaymentTransaction) error
	getTransactionsByOrder func(ctx context.Context, orderID uuid.UUID) ([]PaymentTransaction, error)
}

func (m *mockTxSvc) CreateTransaction(ctx context.Context, t *PaymentTransaction) error {
	if m.createTransaction != nil {
		return m.createTransaction(ctx, t)
	}
	return nil
}
func (m *mockTxSvc) GetTransactionsByOrderID(ctx context.Context, orderID uuid.UUID) ([]PaymentTransaction, error) {
	if m.getTransactionsByOrder != nil {
		return m.getTransactionsByOrder(ctx, orderID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// ListTransactionsByOrder
// ---------------------------------------------------------------------------

func TestHandler_ListTransactionsByOrder_InvalidID(t *testing.T) {
	h := NewHandler(&mockMethodSvc{}, &mockTxSvc{}, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/orders/not-a-uuid/transactions", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", "not-a-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.ListTransactionsByOrder(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_ListTransactionsByOrder_Success(t *testing.T) {
	orderID := uuid.New()
	txID := uuid.New()
	pmID := uuid.New()
	now := time.Now()

	txSvc := &mockTxSvc{
		getTransactionsByOrder: func(_ context.Context, id uuid.UUID) ([]PaymentTransaction, error) {
			if id != orderID {
				t.Errorf("expected orderID %s, got %s", orderID, id)
			}
			return []PaymentTransaction{
				{
					ID:                txID,
					OrderID:           orderID,
					PaymentMethodID:   pmID,
					Status:            "completed",
					Currency:          "EUR",
					Amount:            4999,
					ProviderReference: "pi_abc123",
					CreatedAt:         now,
				},
			}, nil
		},
	}

	h := NewHandler(&mockMethodSvc{}, txSvc, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID.String()+"/transactions", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.ListTransactionsByOrder(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp apiResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Meta == nil {
		t.Fatal("expected meta in response")
	}
	if resp.Meta.Total != 1 {
		t.Errorf("expected total=1, got %d", resp.Meta.Total)
	}

	// Data should be a list
	dataSlice, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("expected data to be a slice, got %T", resp.Data)
	}
	if len(dataSlice) != 1 {
		t.Errorf("expected 1 transaction, got %d", len(dataSlice))
	}
}

func TestHandler_ListTransactionsByOrder_EmptyList(t *testing.T) {
	h := NewHandler(&mockMethodSvc{}, &mockTxSvc{}, zerolog.Nop())

	orderID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID.String()+"/transactions", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.ListTransactionsByOrder(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp apiResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	dataSlice, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("expected data to be a slice, got %T", resp.Data)
	}
	if len(dataSlice) != 0 {
		t.Errorf("expected 0 transactions, got %d", len(dataSlice))
	}
}

func TestHandler_ListTransactionsByOrder_ServiceError(t *testing.T) {
	txSvc := &mockTxSvc{
		getTransactionsByOrder: func(_ context.Context, _ uuid.UUID) ([]PaymentTransaction, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewHandler(&mockMethodSvc{}, txSvc, zerolog.Nop())

	orderID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID.String()+"/transactions", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.ListTransactionsByOrder(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

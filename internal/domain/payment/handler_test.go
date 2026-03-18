package payment

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/internal/auth"
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
	h := NewHandler(&mockMethodSvc{}, &mockTxSvc{}, nil, zerolog.Nop())

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

	h := NewHandler(&mockMethodSvc{}, txSvc, nil, zerolog.Nop())

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
	h := NewHandler(&mockMethodSvc{}, &mockTxSvc{}, nil, zerolog.Nop())

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
	h := NewHandler(&mockMethodSvc{}, txSvc, nil, zerolog.Nop())

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

// ---------------------------------------------------------------------------
// ListTransactionsByOrderStore — ownership checks
// ---------------------------------------------------------------------------

func TestHandler_ListTransactionsByOrderStore_AuthenticatedOwner(t *testing.T) {
	orderID := uuid.New()
	customerID := uuid.New()

	ownershipFn := OrderOwnershipFn(func(_ context.Context, id uuid.UUID) (*uuid.UUID, string, error) {
		if id != orderID {
			t.Errorf("expected orderID %s, got %s", orderID, id)
		}
		return &customerID, "", nil
	})

	txSvc := &mockTxSvc{
		getTransactionsByOrder: func(_ context.Context, id uuid.UUID) ([]PaymentTransaction, error) {
			return []PaymentTransaction{
				{ID: uuid.New(), OrderID: id, Status: "completed", Amount: 4999},
			}, nil
		},
	}

	h := NewHandler(&mockMethodSvc{}, txSvc, ownershipFn, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID.String()+"/transactions", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = auth.WithUserID(ctx, customerID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.ListTransactionsByOrderStore(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp apiResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Meta == nil || resp.Meta.Total != 1 {
		t.Errorf("expected total=1, got %v", resp.Meta)
	}
}

func TestHandler_ListTransactionsByOrderStore_WrongCustomer(t *testing.T) {
	orderID := uuid.New()
	ownerID := uuid.New()
	attackerID := uuid.New()

	ownershipFn := OrderOwnershipFn(func(_ context.Context, _ uuid.UUID) (*uuid.UUID, string, error) {
		return &ownerID, "", nil
	})

	h := NewHandler(&mockMethodSvc{}, &mockTxSvc{}, ownershipFn, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID.String()+"/transactions", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = auth.WithUserID(ctx, attackerID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.ListTransactionsByOrderStore(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestHandler_ListTransactionsByOrderStore_GuestValidTokenCookie(t *testing.T) {
	orderID := uuid.New()
	guestToken := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	ownershipFn := OrderOwnershipFn(func(_ context.Context, _ uuid.UUID) (*uuid.UUID, string, error) {
		return nil, guestToken, nil
	})

	txSvc := &mockTxSvc{
		getTransactionsByOrder: func(_ context.Context, _ uuid.UUID) ([]PaymentTransaction, error) {
			return []PaymentTransaction{}, nil
		},
	}

	h := NewHandler(&mockMethodSvc{}, txSvc, ownershipFn, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID.String()+"/transactions", nil)
	req.AddCookie(&http.Cookie{Name: "stoa_guest_token", Value: guestToken})
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.ListTransactionsByOrderStore(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandler_ListTransactionsByOrderStore_GuestValidTokenQueryParam(t *testing.T) {
	orderID := uuid.New()
	guestToken := "secret-guest-token-123"

	ownershipFn := OrderOwnershipFn(func(_ context.Context, _ uuid.UUID) (*uuid.UUID, string, error) {
		return nil, guestToken, nil
	})

	txSvc := &mockTxSvc{
		getTransactionsByOrder: func(_ context.Context, _ uuid.UUID) ([]PaymentTransaction, error) {
			return []PaymentTransaction{}, nil
		},
	}

	h := NewHandler(&mockMethodSvc{}, txSvc, ownershipFn, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID.String()+"/transactions?guest_token="+guestToken, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.ListTransactionsByOrderStore(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandler_ListTransactionsByOrderStore_GuestInvalidToken(t *testing.T) {
	orderID := uuid.New()

	ownershipFn := OrderOwnershipFn(func(_ context.Context, _ uuid.UUID) (*uuid.UUID, string, error) {
		return nil, "real-token", nil
	})

	h := NewHandler(&mockMethodSvc{}, &mockTxSvc{}, ownershipFn, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID.String()+"/transactions?guest_token=wrong-token", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.ListTransactionsByOrderStore(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestHandler_ListTransactionsByOrderStore_GuestNoToken(t *testing.T) {
	orderID := uuid.New()

	ownershipFn := OrderOwnershipFn(func(_ context.Context, _ uuid.UUID) (*uuid.UUID, string, error) {
		return nil, "real-token", nil
	})

	h := NewHandler(&mockMethodSvc{}, &mockTxSvc{}, ownershipFn, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID.String()+"/transactions", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.ListTransactionsByOrderStore(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestHandler_List_ServiceError_NoInfoDisclosure(t *testing.T) {
	methodSvc := &mockMethodSvc{
		list: func(_ context.Context, _ PaymentMethodFilter) ([]PaymentMethod, int, error) {
			return nil, 0, errors.New("pq: relation \"payment_methods\" does not exist")
		},
	}
	h := NewHandler(methodSvc, &mockTxSvc{}, nil, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/payment-methods", nil)
	w := httptest.NewRecorder()
	h.list(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "an unexpected error occurred") {
		t.Errorf("expected generic error message in response body, got: %s", body)
	}
	if strings.Contains(body, "payment_methods") {
		t.Errorf("response body must not contain internal error details, got: %s", body)
	}
}

func TestHandler_ListTransactionsByOrderStore_OrderNotFound(t *testing.T) {
	ownershipFn := OrderOwnershipFn(func(_ context.Context, _ uuid.UUID) (*uuid.UUID, string, error) {
		return nil, "", errors.New("order not found")
	})

	h := NewHandler(&mockMethodSvc{}, &mockTxSvc{}, ownershipFn, zerolog.Nop())

	orderID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID.String()+"/transactions", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("orderID", orderID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.ListTransactionsByOrderStore(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

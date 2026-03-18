package cart

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/internal/auth"
	"github.com/stoa-hq/stoa/pkg/sdk"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newTestHandler(repo CartRepository) *Handler {
	svc := NewCartService(repo, nil, sdk.NewHookRegistry(), zerolog.Nop())
	return NewHandler(svc, zerolog.Nop())
}

func withChiCartParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func withChiCartParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func withAuthContext(r *http.Request, userID uuid.UUID) *http.Request {
	ctx := auth.WithUserID(r.Context(), userID)
	return r.WithContext(ctx)
}

// ---------------------------------------------------------------------------
// verifyCartOwnership — authenticated customer
// ---------------------------------------------------------------------------

func TestGetCart_OwnerAccess(t *testing.T) {
	customerID := uuid.New()
	cartID := uuid.New()

	repo := &mockCartRepo{
		findByID: func(_ context.Context, id uuid.UUID) (*Cart, error) {
			return &Cart{ID: id, CustomerID: &customerID, SessionID: "s1"}, nil
		},
	}
	h := newTestHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/cart/"+cartID.String(), nil)
	req = withChiCartParam(req, "id", cartID.String())
	req = withAuthContext(req, customerID)

	w := httptest.NewRecorder()
	h.handleGetCart(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", w.Code)
	}
}

func TestGetCart_WrongCustomer_Forbidden(t *testing.T) {
	ownerID := uuid.New()
	attackerID := uuid.New()
	cartID := uuid.New()

	repo := &mockCartRepo{
		findByID: func(_ context.Context, id uuid.UUID) (*Cart, error) {
			return &Cart{ID: id, CustomerID: &ownerID, SessionID: "s1"}, nil
		},
	}
	h := newTestHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/cart/"+cartID.String(), nil)
	req = withChiCartParam(req, "id", cartID.String())
	req = withAuthContext(req, attackerID)

	w := httptest.NewRecorder()
	h.handleGetCart(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status: got %d, want 403", w.Code)
	}
}

// ---------------------------------------------------------------------------
// verifyCartOwnership — guest session
// ---------------------------------------------------------------------------

func TestGetCart_GuestWithCorrectSession(t *testing.T) {
	cartID := uuid.New()
	sessionID := "guest-session-123"

	repo := &mockCartRepo{
		findByID: func(_ context.Context, id uuid.UUID) (*Cart, error) {
			return &Cart{ID: id, SessionID: sessionID}, nil
		},
	}
	h := newTestHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/cart/"+cartID.String(), nil)
	req = withChiCartParam(req, "id", cartID.String())
	req.Header.Set("X-Session-ID", sessionID)

	w := httptest.NewRecorder()
	h.handleGetCart(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", w.Code)
	}
}

func TestGetCart_GuestWrongSession_Forbidden(t *testing.T) {
	cartID := uuid.New()

	repo := &mockCartRepo{
		findByID: func(_ context.Context, id uuid.UUID) (*Cart, error) {
			return &Cart{ID: id, SessionID: "real-session"}, nil
		},
	}
	h := newTestHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/cart/"+cartID.String(), nil)
	req = withChiCartParam(req, "id", cartID.String())
	req.Header.Set("X-Session-ID", "wrong-session")

	w := httptest.NewRecorder()
	h.handleGetCart(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status: got %d, want 403", w.Code)
	}
}

func TestGetCart_GuestNoSession_Forbidden(t *testing.T) {
	cartID := uuid.New()

	repo := &mockCartRepo{
		findByID: func(_ context.Context, id uuid.UUID) (*Cart, error) {
			return &Cart{ID: id, SessionID: "real-session"}, nil
		},
	}
	h := newTestHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/cart/"+cartID.String(), nil)
	req = withChiCartParam(req, "id", cartID.String())
	// No X-Session-ID header

	w := httptest.NewRecorder()
	h.handleGetCart(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status: got %d, want 403", w.Code)
	}
}

// ---------------------------------------------------------------------------
// handleAddItem — ownership check
// ---------------------------------------------------------------------------

func TestAddItem_WrongCustomer_Forbidden(t *testing.T) {
	ownerID := uuid.New()
	attackerID := uuid.New()
	cartID := uuid.New()

	repo := &mockCartRepo{
		findByID: func(_ context.Context, id uuid.UUID) (*Cart, error) {
			return &Cart{ID: id, CustomerID: &ownerID}, nil
		},
	}
	h := newTestHandler(repo)

	body, _ := json.Marshal(AddItemRequest{
		ProductID: uuid.New(),
		Quantity:  1,
	})
	req := httptest.NewRequest(http.MethodPost, "/cart/"+cartID.String()+"/items", bytes.NewReader(body))
	req = withChiCartParam(req, "id", cartID.String())
	req = withAuthContext(req, attackerID)

	w := httptest.NewRecorder()
	h.handleAddItem(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status: got %d, want 403", w.Code)
	}
}

// ---------------------------------------------------------------------------
// handleUpdateItem — ownership check
// ---------------------------------------------------------------------------

func TestUpdateItem_WrongCustomer_Forbidden(t *testing.T) {
	ownerID := uuid.New()
	attackerID := uuid.New()
	cartID := uuid.New()
	itemID := uuid.New()

	repo := &mockCartRepo{
		findByID: func(_ context.Context, id uuid.UUID) (*Cart, error) {
			return &Cart{ID: id, CustomerID: &ownerID}, nil
		},
	}
	h := newTestHandler(repo)

	body, _ := json.Marshal(UpdateItemRequest{Quantity: 5})
	req := httptest.NewRequest(http.MethodPut, "/cart/"+cartID.String()+"/items/"+itemID.String(), bytes.NewReader(body))
	req = withChiCartParams(req, map[string]string{"id": cartID.String(), "itemId": itemID.String()})
	req = withAuthContext(req, attackerID)

	w := httptest.NewRecorder()
	h.handleUpdateItem(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status: got %d, want 403", w.Code)
	}
}

// ---------------------------------------------------------------------------
// handleRemoveItem — ownership check
// ---------------------------------------------------------------------------

func TestRemoveItem_WrongCustomer_Forbidden(t *testing.T) {
	ownerID := uuid.New()
	attackerID := uuid.New()
	cartID := uuid.New()
	itemID := uuid.New()

	repo := &mockCartRepo{
		findByID: func(_ context.Context, id uuid.UUID) (*Cart, error) {
			return &Cart{ID: id, CustomerID: &ownerID}, nil
		},
	}
	h := newTestHandler(repo)

	req := httptest.NewRequest(http.MethodDelete, "/cart/"+cartID.String()+"/items/"+itemID.String(), nil)
	req = withChiCartParams(req, map[string]string{"id": cartID.String(), "itemId": itemID.String()})
	req = withAuthContext(req, attackerID)

	w := httptest.NewRecorder()
	h.handleRemoveItem(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status: got %d, want 403", w.Code)
	}
}

// ---------------------------------------------------------------------------
// handleCreateCart — binds customer_id from auth context
// ---------------------------------------------------------------------------

func TestCreateCart_AuthenticatedUser_SetsCustomerID(t *testing.T) {
	customerID := uuid.New()
	var capturedCustomerID *uuid.UUID

	repo := &mockCartRepo{
		create: func(_ context.Context, c *Cart) error {
			capturedCustomerID = c.CustomerID
			return nil
		},
	}
	h := newTestHandler(repo)

	body, _ := json.Marshal(CreateCartRequest{Currency: "EUR", SessionID: "s1"})
	req := httptest.NewRequest(http.MethodPost, "/cart", bytes.NewReader(body))
	req = withAuthContext(req, customerID)

	w := httptest.NewRecorder()
	h.handleCreateCart(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201", w.Code)
	}
	if capturedCustomerID == nil || *capturedCustomerID != customerID {
		t.Errorf("customer_id: got %v, want %s", capturedCustomerID, customerID)
	}
}

func TestCreateCart_Guest_NilCustomerID(t *testing.T) {
	var capturedCustomerID *uuid.UUID

	repo := &mockCartRepo{
		create: func(_ context.Context, c *Cart) error {
			capturedCustomerID = c.CustomerID
			return nil
		},
	}
	h := newTestHandler(repo)

	body, _ := json.Marshal(CreateCartRequest{Currency: "USD", SessionID: "guest-1"})
	req := httptest.NewRequest(http.MethodPost, "/cart", bytes.NewReader(body))

	w := httptest.NewRecorder()
	h.handleCreateCart(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201", w.Code)
	}
	if capturedCustomerID != nil {
		t.Errorf("customer_id should be nil for guest, got %v", capturedCustomerID)
	}
}

// ---------------------------------------------------------------------------
// Authenticated user trying to access a guest cart (no customer_id)
// ---------------------------------------------------------------------------

func TestGetCart_AuthenticatedUser_GuestCart_Forbidden(t *testing.T) {
	cartID := uuid.New()

	repo := &mockCartRepo{
		findByID: func(_ context.Context, id uuid.UUID) (*Cart, error) {
			return &Cart{ID: id, CustomerID: nil, SessionID: "guest-session"}, nil
		},
	}
	h := newTestHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/cart/"+cartID.String(), nil)
	req = withChiCartParam(req, "id", cartID.String())
	req = withAuthContext(req, uuid.New())

	w := httptest.NewRecorder()
	h.handleGetCart(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status: got %d, want 403", w.Code)
	}
}

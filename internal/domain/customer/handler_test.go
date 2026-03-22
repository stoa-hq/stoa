package customer

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"

	"github.com/google/uuid"

	"github.com/stoa-hq/stoa/internal/auth"
	"github.com/stoa-hq/stoa/pkg/sdk"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newTestCustomerHandler(repo CustomerRepository) *Handler {
	svc := NewCustomerService(repo, sdk.NewHookRegistry(), zerolog.Nop())
	return NewHandler(svc, validator.New(), zerolog.Nop())
}

// ---------------------------------------------------------------------------
// GET /customers — error information disclosure
// ---------------------------------------------------------------------------

func TestHandler_AdminList_ErrorDoesNotLeakInternalDetails(t *testing.T) {
	internalMsg := "pq: relation \"customers\" does not exist"
	repo := &mockCustomerRepo{
		findAll: func(_ context.Context, _ CustomerFilter) ([]Customer, int, error) {
			return nil, 0, errors.New(internalMsg)
		},
	}

	r := chi.NewRouter()
	h := newTestCustomerHandler(repo)
	h.RegisterAdminRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/customers", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want 500", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "an unexpected error occurred") {
		t.Errorf("response should contain generic error message, got: %s", body)
	}
	if strings.Contains(body, internalMsg) {
		t.Errorf("response must NOT contain internal error detail %q, got: %s", internalMsg, body)
	}
}

// ---------------------------------------------------------------------------
// customerIDFromContext — auth.UserID integration
// ---------------------------------------------------------------------------

func TestHandler_customerIDFromContext_WithValidID(t *testing.T) {
	h := newTestCustomerHandler(&mockCustomerRepo{})
	expectedID := uuid.New()

	ctx := auth.WithUserID(t.Context(), expectedID)
	r := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	id, ok := h.customerIDFromContext(w, r)
	if !ok {
		t.Fatal("expected ok=true, got false")
	}
	if id != expectedID {
		t.Fatalf("expected %s, got %s", expectedID, id)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestHandler_customerIDFromContext_WithoutID(t *testing.T) {
	h := newTestCustomerHandler(&mockCustomerRepo{})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	id, ok := h.customerIDFromContext(w, r)
	if ok {
		t.Fatal("expected ok=false, got true")
	}
	if id != uuid.Nil {
		t.Fatalf("expected uuid.Nil, got %s", id)
	}
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

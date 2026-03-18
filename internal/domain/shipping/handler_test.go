package shipping

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ---------------------------------------------------------------------------
// Mock service for handler tests
// ---------------------------------------------------------------------------

type mockShippingSvc struct {
	list    func(ctx context.Context, f ShippingMethodFilter) ([]ShippingMethod, int, error)
	create  func(ctx context.Context, m *ShippingMethod) error
	getByID func(ctx context.Context, id uuid.UUID) (*ShippingMethod, error)
	update  func(ctx context.Context, m *ShippingMethod) error
	delete  func(ctx context.Context, id uuid.UUID) error
}

func (m *mockShippingSvc) List(ctx context.Context, f ShippingMethodFilter) ([]ShippingMethod, int, error) {
	if m.list != nil {
		return m.list(ctx, f)
	}
	return nil, 0, nil
}
func (m *mockShippingSvc) Create(ctx context.Context, sm *ShippingMethod) error {
	if m.create != nil {
		return m.create(ctx, sm)
	}
	return nil
}
func (m *mockShippingSvc) GetByID(ctx context.Context, id uuid.UUID) (*ShippingMethod, error) {
	if m.getByID != nil {
		return m.getByID(ctx, id)
	}
	return nil, ErrNotFound
}
func (m *mockShippingSvc) Update(ctx context.Context, sm *ShippingMethod) error {
	if m.update != nil {
		return m.update(ctx, sm)
	}
	return nil
}
func (m *mockShippingSvc) Delete(ctx context.Context, id uuid.UUID) error {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// List — error information disclosure
// ---------------------------------------------------------------------------

func TestHandler_List_ServiceError_NoInfoDisclosure(t *testing.T) {
	svc := &mockShippingSvc{
		list: func(_ context.Context, _ ShippingMethodFilter) ([]ShippingMethod, int, error) {
			return nil, 0, errors.New("pq: relation \"shipping_methods\" does not exist")
		},
	}
	h := NewHandler(svc, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/shipping-methods", nil)
	w := httptest.NewRecorder()
	h.list(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "an unexpected error occurred") {
		t.Errorf("expected generic error message in response body, got: %s", body)
	}
	if strings.Contains(body, "shipping_methods") {
		t.Errorf("response body must not contain internal error details, got: %s", body)
	}
}

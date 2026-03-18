package discount

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

// mockDiscountService implements DiscountService with optional function fields.
type mockDiscountService struct {
	listFn          func(ctx context.Context, filter DiscountFilter) ([]Discount, int, error)
	createFn        func(ctx context.Context, d *Discount) error
	getByIDFn       func(ctx context.Context, id uuid.UUID) (*Discount, error)
	updateFn        func(ctx context.Context, d *Discount) error
	deleteFn        func(ctx context.Context, id uuid.UUID) error
	validateCodeFn  func(ctx context.Context, code string, orderTotal int) (*Discount, error)
	applyDiscountFn func(ctx context.Context, id uuid.UUID) error
}

var errMockDefault = errors.New("mock: not implemented")

func (m *mockDiscountService) List(ctx context.Context, filter DiscountFilter) ([]Discount, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, 0, errMockDefault
}

func (m *mockDiscountService) Create(ctx context.Context, d *Discount) error {
	if m.createFn != nil {
		return m.createFn(ctx, d)
	}
	return errMockDefault
}

func (m *mockDiscountService) GetByID(ctx context.Context, id uuid.UUID) (*Discount, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, errMockDefault
}

func (m *mockDiscountService) Update(ctx context.Context, d *Discount) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, d)
	}
	return errMockDefault
}

func (m *mockDiscountService) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return errMockDefault
}

func (m *mockDiscountService) ValidateCode(ctx context.Context, code string, orderTotal int) (*Discount, error) {
	if m.validateCodeFn != nil {
		return m.validateCodeFn(ctx, code, orderTotal)
	}
	return nil, errMockDefault
}

func (m *mockDiscountService) ApplyDiscount(ctx context.Context, id uuid.UUID) error {
	if m.applyDiscountFn != nil {
		return m.applyDiscountFn(ctx, id)
	}
	return errMockDefault
}

func TestHandler_List_ServerError_GenericMessage(t *testing.T) {
	internalErrMsg := `pq: relation "discounts" does not exist`

	svc := &mockDiscountService{
		listFn: func(_ context.Context, _ DiscountFilter) ([]Discount, int, error) {
			return nil, 0, errors.New(internalErrMsg)
		},
	}

	h := NewHandler(svc, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/discounts", nil)
	w := httptest.NewRecorder()

	h.list(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}

	body := w.Body.String()
	if !strings.Contains(body, "an unexpected error occurred") {
		t.Errorf("response body should contain generic error message, got: %s", body)
	}
	if strings.Contains(body, internalErrMsg) {
		t.Errorf("response body must not leak internal error details, got: %s", body)
	}
}

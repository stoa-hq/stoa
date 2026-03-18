package tax

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

// mockTaxService implements TaxService with optional function fields.
type mockTaxService struct {
	listFn    func(ctx context.Context, filter TaxRuleFilter) ([]TaxRule, int, error)
	createFn  func(ctx context.Context, t *TaxRule) error
	getByIDFn func(ctx context.Context, id uuid.UUID) (*TaxRule, error)
	updateFn  func(ctx context.Context, t *TaxRule) error
	deleteFn  func(ctx context.Context, id uuid.UUID) error
}

var errMockDefault = errors.New("mock: not implemented")

func (m *mockTaxService) List(ctx context.Context, filter TaxRuleFilter) ([]TaxRule, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, 0, errMockDefault
}

func (m *mockTaxService) Create(ctx context.Context, t *TaxRule) error {
	if m.createFn != nil {
		return m.createFn(ctx, t)
	}
	return errMockDefault
}

func (m *mockTaxService) GetByID(ctx context.Context, id uuid.UUID) (*TaxRule, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, errMockDefault
}

func (m *mockTaxService) Update(ctx context.Context, t *TaxRule) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, t)
	}
	return errMockDefault
}

func (m *mockTaxService) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return errMockDefault
}

func TestHandler_List_ServerError_GenericMessage(t *testing.T) {
	internalErrMsg := `pq: relation "tax_rules" does not exist`

	svc := &mockTaxService{
		listFn: func(_ context.Context, _ TaxRuleFilter) ([]TaxRule, int, error) {
			return nil, 0, errors.New(internalErrMsg)
		},
	}

	h := NewHandler(svc, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/tax-rules", nil)
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

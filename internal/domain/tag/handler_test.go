package tag

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

// mockTagService implements TagService with optional function fields.
type mockTagService struct {
	listFn    func(ctx context.Context, filter TagFilter) ([]Tag, int, error)
	createFn  func(ctx context.Context, t *Tag) error
	getByIDFn func(ctx context.Context, id uuid.UUID) (*Tag, error)
	updateFn  func(ctx context.Context, t *Tag) error
	deleteFn  func(ctx context.Context, id uuid.UUID) error
}

var errMockDefault = errors.New("mock: not implemented")

func (m *mockTagService) List(ctx context.Context, filter TagFilter) ([]Tag, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, 0, errMockDefault
}

func (m *mockTagService) Create(ctx context.Context, t *Tag) error {
	if m.createFn != nil {
		return m.createFn(ctx, t)
	}
	return errMockDefault
}

func (m *mockTagService) GetByID(ctx context.Context, id uuid.UUID) (*Tag, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, errMockDefault
}

func (m *mockTagService) Update(ctx context.Context, t *Tag) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, t)
	}
	return errMockDefault
}

func (m *mockTagService) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return errMockDefault
}

func TestHandler_List_ServerError_GenericMessage(t *testing.T) {
	internalErrMsg := `pq: relation "tags" does not exist`

	svc := &mockTagService{
		listFn: func(_ context.Context, _ TagFilter) ([]Tag, int, error) {
			return nil, 0, errors.New(internalErrMsg)
		},
	}

	h := NewHandler(svc, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/tags", nil)
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

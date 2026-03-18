package media

import (
	"context"
	"errors"
	"io"
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

type mockMediaSvc struct {
	list    func(ctx context.Context, f MediaFilter) ([]Media, int, error)
	upload  func(ctx context.Context, filename, mimeType, altText string, size int64, src io.Reader) (*Media, error)
	getByID func(ctx context.Context, id uuid.UUID) (*Media, error)
	delete  func(ctx context.Context, id uuid.UUID) error
}

func (m *mockMediaSvc) List(ctx context.Context, f MediaFilter) ([]Media, int, error) {
	if m.list != nil {
		return m.list(ctx, f)
	}
	return nil, 0, nil
}
func (m *mockMediaSvc) Upload(ctx context.Context, filename, mimeType, altText string, size int64, src io.Reader) (*Media, error) {
	if m.upload != nil {
		return m.upload(ctx, filename, mimeType, altText, size, src)
	}
	return nil, nil
}
func (m *mockMediaSvc) GetByID(ctx context.Context, id uuid.UUID) (*Media, error) {
	if m.getByID != nil {
		return m.getByID(ctx, id)
	}
	return nil, ErrNotFound
}
func (m *mockMediaSvc) Delete(ctx context.Context, id uuid.UUID) error {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// List — error information disclosure
// ---------------------------------------------------------------------------

func TestHandler_List_ServiceError_NoInfoDisclosure(t *testing.T) {
	svc := &mockMediaSvc{
		list: func(_ context.Context, _ MediaFilter) ([]Media, int, error) {
			return nil, 0, errors.New("pq: relation \"media\" does not exist")
		},
	}
	h := NewHandler(svc, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/media", nil)
	w := httptest.NewRecorder()
	h.list(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "an unexpected error occurred") {
		t.Errorf("expected generic error message in response body, got: %s", body)
	}
	if strings.Contains(body, "pq:") {
		t.Errorf("response body must not contain internal error details, got: %s", body)
	}
}

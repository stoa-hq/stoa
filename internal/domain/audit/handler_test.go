package audit

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

// mockAuditService implements AuditService with optional function fields.
type mockAuditService struct {
	logFn  func(ctx context.Context, a *AuditLog) error
	listFn func(ctx context.Context, filter AuditFilter) ([]AuditLog, int, error)
}

var errMockDefault = errors.New("mock: not implemented")

func (m *mockAuditService) Log(ctx context.Context, a *AuditLog) error {
	if m.logFn != nil {
		return m.logFn(ctx, a)
	}
	return errMockDefault
}

func (m *mockAuditService) List(ctx context.Context, filter AuditFilter) ([]AuditLog, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, 0, errMockDefault
}

func TestHandler_List_ServerError_GenericMessage(t *testing.T) {
	internalErrMsg := `pq: relation "audit_logs" does not exist`

	svc := &mockAuditService{
		listFn: func(_ context.Context, _ AuditFilter) ([]AuditLog, int, error) {
			return nil, 0, errors.New(internalErrMsg)
		},
	}

	h := NewHandler(svc, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/audit-log", nil)
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

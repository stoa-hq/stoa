package category

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"

	"github.com/stoa-hq/stoa/pkg/sdk"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newTestCategoryHandler(repo CategoryRepository) *Handler {
	svc := NewService(repo, sdk.NewHookRegistry(), zerolog.Nop())
	return NewHandler(svc, zerolog.Nop())
}

// ---------------------------------------------------------------------------
// GET /admin/categories — error information disclosure
// ---------------------------------------------------------------------------

func TestHandler_List_ErrorDoesNotLeakInternalDetails(t *testing.T) {
	internalMsg := "pq: relation \"categories\" does not exist"
	repo := &mockCategoryRepo{
		findAll: func(_ context.Context, _ CategoryFilter) ([]Category, int, error) {
			return nil, 0, errors.New(internalMsg)
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/categories", nil)
	w := httptest.NewRecorder()

	newTestCategoryHandler(repo).List(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want 500", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "an unexpected error occurred") {
		t.Errorf("response should contain generic error message, got: %s", body)
	}
	if strings.Contains(body, internalMsg) {
		t.Errorf("response must NOT contain internal error detail %q, got: %s", internalMsg, body)
	}
}

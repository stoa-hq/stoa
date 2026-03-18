package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
)

func dummyHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func newRateLimitedRouter(limit int, path string) chi.Router {
	r := chi.NewRouter()
	r.With(httprate.LimitByIP(limit, time.Minute)).Post(path, dummyHandler)
	return r
}

func doRequests(t *testing.T, r chi.Router, method, path, remoteAddr string, n int) []*httptest.ResponseRecorder {
	t.Helper()
	results := make([]*httptest.ResponseRecorder, n)
	for i := 0; i < n; i++ {
		req := httptest.NewRequest(method, path, nil)
		req.RemoteAddr = remoteAddr
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		results[i] = w
	}
	return results
}

// ---------------------------------------------------------------------------
// Login: 10 req/min
// ---------------------------------------------------------------------------

func TestEndpointRateLimit_Login(t *testing.T) {
	r := newRateLimitedRouter(10, "/api/v1/auth/login")
	results := doRequests(t, r, http.MethodPost, "/api/v1/auth/login", "192.0.2.1:1234", 11)

	for i := 0; i < 10; i++ {
		if results[i].Code != http.StatusOK {
			t.Errorf("request %d: got %d, want %d", i+1, results[i].Code, http.StatusOK)
		}
	}
	if results[10].Code != http.StatusTooManyRequests {
		t.Errorf("request 11: got %d, want %d", results[10].Code, http.StatusTooManyRequests)
	}
}

// ---------------------------------------------------------------------------
// Register: 5 req/min
// ---------------------------------------------------------------------------

func TestEndpointRateLimit_Register(t *testing.T) {
	r := newRateLimitedRouter(5, "/register")
	results := doRequests(t, r, http.MethodPost, "/register", "192.0.2.1:1234", 6)

	for i := 0; i < 5; i++ {
		if results[i].Code != http.StatusOK {
			t.Errorf("request %d: got %d, want %d", i+1, results[i].Code, http.StatusOK)
		}
	}
	if results[5].Code != http.StatusTooManyRequests {
		t.Errorf("request 6: got %d, want %d", results[5].Code, http.StatusTooManyRequests)
	}
}

// ---------------------------------------------------------------------------
// Checkout: 10 req/min
// ---------------------------------------------------------------------------

func TestEndpointRateLimit_Checkout(t *testing.T) {
	r := newRateLimitedRouter(10, "/checkout")
	results := doRequests(t, r, http.MethodPost, "/checkout", "192.0.2.1:1234", 11)

	for i := 0; i < 10; i++ {
		if results[i].Code != http.StatusOK {
			t.Errorf("request %d: got %d, want %d", i+1, results[i].Code, http.StatusOK)
		}
	}
	if results[10].Code != http.StatusTooManyRequests {
		t.Errorf("request 11: got %d, want %d", results[10].Code, http.StatusTooManyRequests)
	}
}

// ---------------------------------------------------------------------------
// Login limit does NOT affect refresh
// ---------------------------------------------------------------------------

func TestEndpointRateLimit_LoginDoesNotAffectRefresh(t *testing.T) {
	r := chi.NewRouter()
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.With(httprate.LimitByIP(5, time.Minute)).Post("/login", dummyHandler)
		r.Post("/refresh", dummyHandler)
	})

	// Exhaust the login limit.
	doRequests(t, r, http.MethodPost, "/api/v1/auth/login", "192.0.2.1:1234", 6)

	// Refresh must still work.
	results := doRequests(t, r, http.MethodPost, "/api/v1/auth/refresh", "192.0.2.1:1234", 1)
	if results[0].Code != http.StatusOK {
		t.Errorf("refresh after login limit: got %d, want %d", results[0].Code, http.StatusOK)
	}
}

// ---------------------------------------------------------------------------
// Different IPs have independent counters
// ---------------------------------------------------------------------------

func TestEndpointRateLimit_DifferentIPsIndependent(t *testing.T) {
	r := newRateLimitedRouter(2, "/api/v1/auth/login")

	// IP A: exhaust limit
	doRequests(t, r, http.MethodPost, "/api/v1/auth/login", "192.0.2.1:1234", 2)

	// IP B: should still be allowed
	results := doRequests(t, r, http.MethodPost, "/api/v1/auth/login", "192.0.2.2:1234", 1)
	if results[0].Code != http.StatusOK {
		t.Errorf("different IP: got %d, want %d", results[0].Code, http.StatusOK)
	}
}

// ---------------------------------------------------------------------------
// 429 response includes Retry-After header
// ---------------------------------------------------------------------------

func TestEndpointRateLimit_RetryAfterHeader(t *testing.T) {
	r := newRateLimitedRouter(1, "/api/v1/auth/login")
	results := doRequests(t, r, http.MethodPost, "/api/v1/auth/login", "192.0.2.1:1234", 2)

	if results[1].Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", results[1].Code)
	}
	retryAfter := results[1].Header().Get("Retry-After")
	if retryAfter == "" {
		t.Error("429 response missing Retry-After header")
	}
}

// ---------------------------------------------------------------------------
// Guest Order Lookup: 10 req/min
// ---------------------------------------------------------------------------

func TestEndpointRateLimit_GuestOrderLookup(t *testing.T) {
	r := chi.NewRouter()
	r.With(httprate.LimitByIP(10, time.Minute)).
		Get("/api/v1/store/orders/{orderID}/transactions", dummyHandler)

	results := doRequests(t, r, http.MethodGet, "/api/v1/store/orders/00000000-0000-0000-0000-000000000001/transactions", "192.0.2.1:1234", 11)

	for i := 0; i < 10; i++ {
		if results[i].Code != http.StatusOK {
			t.Errorf("request %d: got %d, want %d", i+1, results[i].Code, http.StatusOK)
		}
	}
	if results[10].Code != http.StatusTooManyRequests {
		t.Errorf("request 11: got %d, want %d", results[10].Code, http.StatusTooManyRequests)
	}
}

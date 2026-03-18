package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServeIndex_NonceInCSPHeader(t *testing.T) {
	html := []byte(`<!DOCTYPE html><html><head><script type="module" src="/app.js"></script></head></html>`)
	w := httptest.NewRecorder()

	serveIndex(w, html, defaultCSP)

	cspHeader := w.Header().Get("Content-Security-Policy")

	// Extract script-src directive to verify nonce replaces unsafe-inline there.
	scriptSrc := extractDirective(cspHeader, "script-src")
	if strings.Contains(scriptSrc, "'unsafe-inline'") {
		t.Error("script-src still contains 'unsafe-inline'")
	}

	if !strings.Contains(cspHeader, "'nonce-") {
		t.Error("CSP header does not contain nonce")
	}

	if !strings.Contains(cspHeader, "'strict-dynamic'") {
		t.Error("CSP header does not contain 'strict-dynamic'")
	}

	if strings.Contains(cspHeader, "{{NONCE}}") {
		t.Error("CSP header still contains {{NONCE}} placeholder")
	}
}

// extractDirective returns the value of a CSP directive (e.g. "script-src") from a full CSP string.
func extractDirective(csp, directive string) string {
	idx := strings.Index(csp, directive)
	if idx == -1 {
		return ""
	}
	rest := csp[idx:]
	if end := strings.Index(rest, ";"); end != -1 {
		return rest[:end]
	}
	return rest
}

func TestServeIndex_NonceInHTML(t *testing.T) {
	html := []byte(`<!DOCTYPE html><html><head><script type="module" src="/app.js"></script></head></html>`)
	w := httptest.NewRecorder()

	serveIndex(w, html, defaultCSP)

	body := w.Body.String()
	if !strings.Contains(body, `nonce="`) {
		t.Error("HTML body does not contain nonce attribute on script tags")
	}

	if strings.Contains(body, "<script type") {
		t.Error("HTML contains script tag without nonce")
	}
}

func TestServeIndex_UniqueNoncePerRequest(t *testing.T) {
	html := []byte(`<!DOCTYPE html><html><head><script src="/app.js"></script></head></html>`)

	w1 := httptest.NewRecorder()
	serveIndex(w1, html, defaultCSP)
	csp1 := w1.Header().Get("Content-Security-Policy")

	w2 := httptest.NewRecorder()
	serveIndex(w2, html, defaultCSP)
	csp2 := w2.Header().Get("Content-Security-Policy")

	if csp1 == csp2 {
		t.Error("two requests produced identical CSP headers (same nonce)")
	}
}

func TestServeIndex_NonceConsistentBetweenHeaderAndHTML(t *testing.T) {
	html := []byte(`<!DOCTYPE html><html><head><script src="/app.js"></script></head></html>`)
	w := httptest.NewRecorder()

	serveIndex(w, html, defaultCSP)

	cspHeader := w.Header().Get("Content-Security-Policy")
	body := w.Body.String()

	// Extract nonce from CSP header.
	idx := strings.Index(cspHeader, "'nonce-")
	if idx == -1 {
		t.Fatal("no nonce found in CSP header")
	}
	rest := cspHeader[idx+7:] // skip "'nonce-"
	end := strings.Index(rest, "'")
	if end == -1 {
		t.Fatal("malformed nonce in CSP header")
	}
	nonce := rest[:end]

	if !strings.Contains(body, `nonce="`+nonce+`"`) {
		t.Errorf("nonce %q from CSP header not found in HTML body", nonce)
	}
}

func TestHandler_ServesWithNonceCSP(t *testing.T) {
	handler := Handler()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	cspHeader := w.Header().Get("Content-Security-Policy")
	scriptSrc := extractDirective(cspHeader, "script-src")
	if strings.Contains(scriptSrc, "'unsafe-inline'") {
		t.Error("Handler script-src still contains 'unsafe-inline'")
	}
	if !strings.Contains(cspHeader, "'nonce-") {
		t.Error("Handler CSP does not contain nonce")
	}
}

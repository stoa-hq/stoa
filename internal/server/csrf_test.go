package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// ---------------------------------------------------------------------------
// Safe methods — no CSRF check
// ---------------------------------------------------------------------------

func TestCSRF_SafeMethodsDoNotRequireToken(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		t.Run(method, func(t *testing.T) {
			mw := CSRF(false)(http.HandlerFunc(okHandler))
			r := httptest.NewRequest(method, "/", nil)
			w := httptest.NewRecorder()
			mw.ServeHTTP(w, r)
			if w.Code != http.StatusOK {
				t.Errorf("%s: got %d, want %d", method, w.Code, http.StatusOK)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Cookie management
// ---------------------------------------------------------------------------

func TestCSRF_GetSetsCookie(t *testing.T) {
	mw := CSRF(false)(http.HandlerFunc(okHandler))
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, r)

	var found bool
	for _, c := range w.Result().Cookies() {
		if c.Name == csrfCookieName {
			found = true
			if c.Value == "" {
				t.Error("csrf_token cookie value must not be empty")
			}
		}
	}
	if !found {
		t.Error("csrf_token cookie should be set on first GET request")
	}
}

func TestCSRF_ExistingCookieIsReused(t *testing.T) {
	mw := CSRF(false)(http.HandlerFunc(okHandler))
	existingToken := "existing-token-abcdef"

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: csrfCookieName, Value: existingToken})
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, r)

	// No new csrf_token cookie should appear in the response.
	for _, c := range w.Result().Cookies() {
		if c.Name == csrfCookieName {
			t.Error("should not set a new csrf_token cookie when one already exists")
		}
	}
}

func TestCSRF_SecureFlagPropagated(t *testing.T) {
	mw := CSRF(true)(http.HandlerFunc(okHandler))
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, r)

	for _, c := range w.Result().Cookies() {
		if c.Name == csrfCookieName && !c.Secure {
			t.Error("csrf_token cookie should have Secure=true when secure=true")
		}
	}
}

// ---------------------------------------------------------------------------
// Mutation methods require a valid token
// ---------------------------------------------------------------------------

func TestCSRF_MutationMethodsRequireToken(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			mw := CSRF(false)(http.HandlerFunc(okHandler))
			r := httptest.NewRequest(method, "/", nil)
			w := httptest.NewRecorder()
			mw.ServeHTTP(w, r)
			if w.Code != http.StatusForbidden {
				t.Errorf("%s without token: got %d, want %d", method, w.Code, http.StatusForbidden)
			}
		})
	}
}

func TestCSRF_PostWithInvalidToken(t *testing.T) {
	mw := CSRF(false)(http.HandlerFunc(okHandler))

	// Obtain a real cookie first.
	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getW := httptest.NewRecorder()
	mw.ServeHTTP(getW, getReq)
	cookie := getW.Result().Cookies()[0]

	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.AddCookie(cookie)
	r.Header.Set(csrfHeaderName, "wrong-token")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("POST with invalid token: got %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestCSRF_PostWithValidToken(t *testing.T) {
	mw := CSRF(false)(http.HandlerFunc(okHandler))

	// Obtain a real cookie.
	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getW := httptest.NewRecorder()
	mw.ServeHTTP(getW, getReq)
	cookie := getW.Result().Cookies()[0]

	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.AddCookie(cookie)
	r.Header.Set(csrfHeaderName, cookie.Value)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("POST with valid token: got %d, want %d", w.Code, http.StatusOK)
	}
}

// ---------------------------------------------------------------------------
// Authorization header exemption
// ---------------------------------------------------------------------------

func TestCSRF_BearerTokenExemptsCSRFCheck(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			mw := CSRF(false)(http.HandlerFunc(okHandler))
			r := httptest.NewRequest(method, "/", nil)
			r.Header.Set("Authorization", "Bearer some-jwt-token")
			w := httptest.NewRecorder()
			mw.ServeHTTP(w, r)
			if w.Code != http.StatusOK {
				t.Errorf("%s with Bearer header: got %d, want %d", method, w.Code, http.StatusOK)
			}
		})
	}
}

func TestCSRF_APIKeyExemptsCSRFCheck(t *testing.T) {
	mw := CSRF(false)(http.HandlerFunc(okHandler))
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("Authorization", "ApiKey some-api-key")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("POST with ApiKey header: got %d, want %d", w.Code, http.StatusOK)
	}
}

// ---------------------------------------------------------------------------
// Path exemption
// ---------------------------------------------------------------------------

func TestCSRF_ExemptPrefixSkipsCheck(t *testing.T) {
	mw := CSRF(false, "/hooks/")(http.HandlerFunc(okHandler))
	r := httptest.NewRequest(http.MethodPost, "/hooks/something", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("POST to exempt prefix path: got %d, want %d", w.Code, http.StatusOK)
	}
}

func TestCSRF_NonExemptPathStillChecked(t *testing.T) {
	mw := CSRF(false)(http.HandlerFunc(okHandler))
	r := httptest.NewRequest(http.MethodPost, "/api/v1/store/cart", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("POST to non-exempt path: got %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestCSRF_PluginWebhookPathExempt(t *testing.T) {
	for _, path := range []string{
		"/plugins/stripe/webhooks/event",
		"/plugins/n8n/webhooks/trigger",
		"/plugins/paypal/webhooks",
	} {
		t.Run(path, func(t *testing.T) {
			mw := CSRF(false)(http.HandlerFunc(okHandler))
			r := httptest.NewRequest(http.MethodPost, path, nil)
			w := httptest.NewRecorder()
			mw.ServeHTTP(w, r)
			if w.Code != http.StatusOK {
				t.Errorf("POST to plugin webhook path %s: got %d, want %d", path, w.Code, http.StatusOK)
			}
		})
	}
}

func TestCSRF_PluginNonWebhookPathRequiresToken(t *testing.T) {
	for _, path := range []string{
		"/plugins/stripe/admin/settings",
		"/plugins/stripe/store/products",
		"/plugins/stripe/assets/script.js",
	} {
		t.Run(path, func(t *testing.T) {
			mw := CSRF(false)(http.HandlerFunc(okHandler))
			r := httptest.NewRequest(http.MethodPost, path, nil)
			w := httptest.NewRecorder()
			mw.ServeHTTP(w, r)
			if w.Code != http.StatusForbidden {
				t.Errorf("POST to non-webhook plugin path %s: got %d, want %d", path, w.Code, http.StatusForbidden)
			}
		})
	}
}

func TestCSRF_PluginRootPathRequiresToken(t *testing.T) {
	mw := CSRF(false)(http.HandlerFunc(okHandler))
	r := httptest.NewRequest(http.MethodPost, "/plugins/stripe", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("POST to plugin root path: got %d, want %d", w.Code, http.StatusForbidden)
	}
}

// ---------------------------------------------------------------------------
// Error response format
// ---------------------------------------------------------------------------

func TestCSRF_ErrorResponseFormat(t *testing.T) {
	mw := CSRF(false)(http.HandlerFunc(okHandler))
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, r)

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: got %q, want application/json", ct)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(resp.Errors) == 0 {
		t.Fatal("expected at least one error in response")
	}
	if resp.Errors[0].Code != "csrf_token_invalid" {
		t.Errorf("error code: got %q, want csrf_token_invalid", resp.Errors[0].Code)
	}
}

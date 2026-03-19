package server

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

const (
	csrfCookieName = "csrf_token"
	csrfHeaderName = "X-CSRF-Token"
	csrfTokenLen   = 32 // bytes → 64 hex chars
)

// CSRF returns a middleware that implements the Double Submit Cookie pattern.
//
// Requests carrying an Authorization header (Bearer / ApiKey) are exempt because
// CSRF attacks cannot inject custom request headers cross-origin.
//
// Paths matching exemptPrefixes are skipped entirely. This is intended for
// plugin webhook endpoints that authenticate via provider signatures (e.g. Stripe HMAC).
//
// For cookie-authenticated requests the middleware:
//   - Sets a csrf_token cookie on first visit (readable by JavaScript, SameSite=Strict).
//   - Validates that state-changing methods (POST, PUT, PATCH, DELETE) include the
//     matching token in the X-CSRF-Token request header.
//
// secure controls whether the cookie carries the Secure flag (set true behind HTTPS).
func CSRF(secure bool, exemptPrefixes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Bearer / ApiKey requests are immune to CSRF by design:
			// cross-origin requests cannot set custom headers.
			if r.Header.Get("Authorization") != "" {
				next.ServeHTTP(w, r)
				return
			}

			// Exempt paths (plugin webhooks) that authenticate via
			// provider-specific signatures rather than CSRF tokens.
			for _, prefix := range exemptPrefixes {
				if strings.HasPrefix(r.URL.Path, prefix) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Plugin webhook paths authenticate via provider signatures
			// (e.g. Stripe HMAC), not cookies or CSRF tokens.
			if isPluginWebhookPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			cookieToken := ensureCSRFCookie(w, r, secure)

			if requiresCSRFCheck(r.Method) {
				headerToken := r.Header.Get(csrfHeaderName)
				if headerToken == "" || headerToken != cookieToken {
					writeJSON(w, http.StatusForbidden, APIResponse{
						Errors: []APIError{{
							Code:   "csrf_token_invalid",
							Detail: "CSRF token missing or invalid",
						}},
					})
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ensureCSRFCookie returns the existing csrf_token cookie value, or generates a new
// token, sets the cookie, and returns that value.
func ensureCSRFCookie(w http.ResponseWriter, r *http.Request, secure bool) string {
	if c, err := r.Cookie(csrfCookieName); err == nil && c.Value != "" {
		return c.Value
	}

	token := newCSRFToken()
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // JavaScript must read it for the Double Submit pattern.
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
	})
	return token
}

func newCSRFToken() string {
	b := make([]byte, csrfTokenLen)
	if _, err := rand.Read(b); err != nil {
		panic("csrf: cannot generate token: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// isPluginWebhookPath returns true for paths matching /plugins/{name}/webhooks/*.
// Only webhook endpoints are exempt from CSRF because they authenticate via
// provider-specific signatures (e.g. Stripe HMAC).
func isPluginWebhookPath(path string) bool {
	if !strings.HasPrefix(path, "/plugins/") {
		return false
	}
	rest := strings.TrimPrefix(path, "/plugins/")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) < 2 {
		return false
	}
	return strings.HasPrefix(parts[1], "webhooks")
}

func requiresCSRFCheck(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

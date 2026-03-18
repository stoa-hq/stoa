package server

import (
	"net/http"
	"strings"
)

// MaxBytes returns middleware that limits the size of incoming request bodies.
// Multipart/form-data requests use uploadLimit; all other methods with a body
// use bodyLimit. GET, HEAD, and OPTIONS requests are not wrapped.
func MaxBytes(bodyLimit, uploadLimit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}

			limit := bodyLimit
			if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
				limit = uploadLimit
			}

			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

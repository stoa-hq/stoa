package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaxBytes(t *testing.T) {
	const bodyLimit = 64
	const uploadLimit = 256

	echo := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := MaxBytes(bodyLimit, uploadLimit)(echo)

	t.Run("GET passes without wrapping", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("HEAD passes without wrapping", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodHead, "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("POST small body OK", func(t *testing.T) {
		body := strings.NewReader(`{"ok": true}`)
		req := httptest.NewRequest(http.MethodPost, "/", body)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("POST body exactly at limit", func(t *testing.T) {
		body := strings.NewReader(strings.Repeat("x", bodyLimit))
		req := httptest.NewRequest(http.MethodPost, "/", body)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("POST body exceeds limit returns 413", func(t *testing.T) {
		body := strings.NewReader(strings.Repeat("x", bodyLimit+1))
		req := httptest.NewRequest(http.MethodPost, "/", body)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected 413, got %d", rr.Code)
		}
	})

	t.Run("PUT body exceeds limit returns 413", func(t *testing.T) {
		body := strings.NewReader(strings.Repeat("x", bodyLimit+1))
		req := httptest.NewRequest(http.MethodPut, "/", body)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected 413, got %d", rr.Code)
		}
	})

	t.Run("multipart uses higher upload limit", func(t *testing.T) {
		// Body larger than bodyLimit but smaller than uploadLimit
		body := strings.NewReader(strings.Repeat("x", bodyLimit+10))
		req := httptest.NewRequest(http.MethodPost, "/", body)
		req.Header.Set("Content-Type", "multipart/form-data; boundary=abc")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("multipart exceeding upload limit returns 413", func(t *testing.T) {
		body := strings.NewReader(strings.Repeat("x", uploadLimit+1))
		req := httptest.NewRequest(http.MethodPost, "/", body)
		req.Header.Set("Content-Type", "multipart/form-data; boundary=abc")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected 413, got %d", rr.Code)
		}
	})
}

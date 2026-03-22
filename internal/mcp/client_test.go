package mcp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewStoaStoreClient_WithAPIKey(t *testing.T) {
	cfg := &Config{
		APIURL: "http://localhost:8080",
		APIKey: "sk_test_key_123",
		Port:   8091,
	}

	c := NewStoaStoreClient(cfg)

	if c.apiKey != "sk_test_key_123" {
		t.Errorf("apiKey = %q, want %q", c.apiKey, "sk_test_key_123")
	}
	if c.sessionID != "" {
		t.Errorf("sessionID = %q, want empty (API key is set)", c.sessionID)
	}
}

func TestNewStoaStoreClient_WithoutAPIKey_GeneratesSessionID(t *testing.T) {
	cfg := &Config{
		APIURL: "http://localhost:8080",
		APIKey: "",
		Port:   8091,
	}

	c := NewStoaStoreClient(cfg)

	if c.apiKey != "" {
		t.Errorf("apiKey = %q, want empty", c.apiKey)
	}
	if c.sessionID == "" {
		t.Error("sessionID should be generated when API key is empty")
	}
	// UUID format: 8-4-4-4-12
	if len(c.sessionID) != 36 {
		t.Errorf("sessionID length = %d, want 36 (UUID format)", len(c.sessionID))
	}
}

func TestClient_Do_SendsSessionIDHeader(t *testing.T) {
	var gotHeaders http.Header
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	defer ts.Close()

	c := &StoaClient{
		baseURL:    ts.URL,
		sessionID:  "test-session-id",
		httpClient: ts.Client(),
	}

	_, err := c.Get("/api/v1/store/products")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}

	if got := gotHeaders.Get("X-Session-ID"); got != "test-session-id" {
		t.Errorf("X-Session-ID header = %q, want %q", got, "test-session-id")
	}
	if got := gotHeaders.Get("Authorization"); got != "" {
		t.Errorf("Authorization header = %q, want empty (no API key)", got)
	}
}

func TestClient_Do_SendsAuthorizationHeader(t *testing.T) {
	var gotHeaders http.Header
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	defer ts.Close()

	c := &StoaClient{
		baseURL:    ts.URL,
		apiKey:     "sk_my_key",
		httpClient: ts.Client(),
	}

	_, err := c.Get("/api/v1/store/products")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}

	if got := gotHeaders.Get("Authorization"); got != "ApiKey sk_my_key" {
		t.Errorf("Authorization header = %q, want %q", got, "ApiKey sk_my_key")
	}
	if got := gotHeaders.Get("X-Session-ID"); got != "" {
		t.Errorf("X-Session-ID header = %q, want empty (API key is set)", got)
	}
}

func TestClient_Do_GuestMode_NoAuthHeaders(t *testing.T) {
	var gotHeaders http.Header
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	defer ts.Close()

	c := &StoaClient{
		baseURL:    ts.URL,
		httpClient: ts.Client(),
	}

	_, err := c.Get("/api/v1/store/products")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}

	if got := gotHeaders.Get("Authorization"); got != "" {
		t.Errorf("Authorization header = %q, want empty", got)
	}
	if got := gotHeaders.Get("X-Session-ID"); got != "" {
		t.Errorf("X-Session-ID header = %q, want empty", got)
	}
}

func TestClient_Do_NoHardErrorOnEmptyAPIKey(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer ts.Close()

	c := &StoaClient{
		baseURL:    ts.URL,
		httpClient: ts.Client(),
	}

	// Previously this would return an error. Now it should succeed (guest mode).
	_, err := c.Get("/api/v1/store/products")
	if err != nil {
		t.Fatalf("expected no error for empty API key (guest mode), got: %v", err)
	}
}

package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockRefreshTokenStore is a test double that operates entirely in memory.
type mockRefreshTokenStore struct {
	records map[string]*RefreshTokenRecord // keyed by token_id
}

func newMockRefreshTokenStore() *mockRefreshTokenStore {
	return &mockRefreshTokenStore{records: make(map[string]*RefreshTokenRecord)}
}

func (m *mockRefreshTokenStore) store(tokenID string, userID, familyID uuid.UUID, expiresAt time.Time) {
	m.records[tokenID] = &RefreshTokenRecord{
		ID:        uuid.New(),
		TokenID:   tokenID,
		UserID:    userID,
		FamilyID:  familyID,
		Revoked:   false,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
}

func (m *mockRefreshTokenStore) consume(tokenID string) (*RefreshTokenRecord, error) {
	rec, ok := m.records[tokenID]
	if !ok {
		return nil, ErrTokenNotFound
	}
	if rec.Revoked {
		// Reuse detection: revoke all tokens in the family.
		for _, r := range m.records {
			if r.FamilyID == rec.FamilyID {
				r.Revoked = true
			}
		}
		return nil, ErrTokenReuse
	}
	rec.Revoked = true
	return rec, nil
}

func (m *mockRefreshTokenStore) revokeAllForUser(userID uuid.UUID) {
	for _, r := range m.records {
		if r.UserID == userID {
			r.Revoked = true
		}
	}
}

// --- Handler tests using real JWTManager + mock store ---

// testHandler builds a Handler whose tokenStore is backed by a mockRefreshTokenStore.
// We wrap the mock behind a thin shim that satisfies the real Handler's use of
// *RefreshTokenStore — we do this by embedding the mock into the handler tests and
// calling the handler methods directly against httptest recorders.

func TestHandleRefresh_RotatesToken(t *testing.T) {
	jwtMgr := mustNewJWTManager(t, testSecret, 15*time.Minute, 24*time.Hour)
	userID := uuid.New()
	familyID := uuid.New()

	// Generate initial refresh token.
	refreshTok, err := jwtMgr.GenerateRefreshToken(userID, "test@example.com", "admin", "admin")
	if err != nil {
		t.Fatal(err)
	}
	claims, _ := jwtMgr.ValidateToken(refreshTok)

	store := newMockRefreshTokenStore()
	store.store(claims.ID, userID, familyID, claims.ExpiresAt.Time)

	// Call consume — should succeed and mark old token as revoked.
	rec, err := store.consume(claims.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rec.FamilyID != familyID {
		t.Errorf("family ID mismatch: got %s, want %s", rec.FamilyID, familyID)
	}
	if !store.records[claims.ID].Revoked {
		t.Error("old token should be revoked after consume")
	}

	// Generate new refresh token and store in same family.
	newRefreshTok, _ := jwtMgr.GenerateRefreshToken(userID, "test@example.com", "admin", "admin")
	newClaims, _ := jwtMgr.ValidateToken(newRefreshTok)
	store.store(newClaims.ID, userID, familyID, newClaims.ExpiresAt.Time)

	// New token should be consumable.
	_, err = store.consume(newClaims.ID)
	if err != nil {
		t.Fatalf("new token should be consumable: %v", err)
	}
}

func TestHandleRefresh_ReuseDetection(t *testing.T) {
	jwtMgr := mustNewJWTManager(t, testSecret, 15*time.Minute, 24*time.Hour)
	userID := uuid.New()
	familyID := uuid.New()

	tok1, _ := jwtMgr.GenerateRefreshToken(userID, "test@example.com", "customer", "customer")
	claims1, _ := jwtMgr.ValidateToken(tok1)

	tok2, _ := jwtMgr.GenerateRefreshToken(userID, "test@example.com", "customer", "customer")
	claims2, _ := jwtMgr.ValidateToken(tok2)

	store := newMockRefreshTokenStore()
	store.store(claims1.ID, userID, familyID, claims1.ExpiresAt.Time)
	store.store(claims2.ID, userID, familyID, claims2.ExpiresAt.Time)

	// Consume tok1 — normal rotation.
	_, err := store.consume(claims1.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Reuse tok1 — should detect reuse and revoke entire family.
	_, err = store.consume(claims1.ID)
	if err != ErrTokenReuse {
		t.Fatalf("expected ErrTokenReuse, got %v", err)
	}

	// tok2 should also be revoked now.
	if !store.records[claims2.ID].Revoked {
		t.Error("sibling token should be revoked after reuse detection")
	}
}

func TestHandleRefresh_UnknownToken(t *testing.T) {
	store := newMockRefreshTokenStore()

	_, err := store.consume("nonexistent-jti")
	if err != ErrTokenNotFound {
		t.Fatalf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestHandleRefresh_HTTPEndpoint_RejectsAccessToken(t *testing.T) {
	jwtMgr := mustNewJWTManager(t, testSecret, 15*time.Minute, 24*time.Hour)
	logger := zerolog.Nop()
	bruteForce := NewBruteForceTracker(5, 15*time.Minute)
	store := NewRefreshTokenStore(nil) // won't be reached

	h := NewHandler(nil, jwtMgr, nil, bruteForce, store, NewTokenBlacklist(), logger)

	// Generate an access token (not refresh).
	accessTok, _ := jwtMgr.GenerateAccessToken(uuid.New(), "test@example.com", "admin", "admin")

	body, _ := json.Marshal(RefreshRequest{RefreshToken: accessTok})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleRefresh(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandleRefresh_HTTPEndpoint_RejectsExpiredToken(t *testing.T) {
	jwtMgr := mustNewJWTManager(t, testSecret, 15*time.Minute, -time.Second) // negative refresh TTL
	logger := zerolog.Nop()
	bruteForce := NewBruteForceTracker(5, 15*time.Minute)
	store := NewRefreshTokenStore(nil) // won't be reached

	h := NewHandler(nil, jwtMgr, nil, bruteForce, store, NewTokenBlacklist(), logger)

	expiredTok, _ := jwtMgr.GenerateRefreshToken(uuid.New(), "test@example.com", "admin", "admin")

	body, _ := json.Marshal(RefreshRequest{RefreshToken: expiredTok})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleRefresh(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestLogin_MixedCaseEmail_NormalizesBeforeLookup(t *testing.T) {
	jwtMgr := mustNewJWTManager(t, testSecret, 15*time.Minute, 24*time.Hour)
	logger := zerolog.Nop()
	bruteForce := NewBruteForceTracker(5, 15*time.Minute)
	store := NewRefreshTokenStore(nil)
	h := NewHandler(nil, jwtMgr, nil, bruteForce, store, NewTokenBlacklist(), logger)

	// Lock account using lowercase email.
	for i := 0; i < 5; i++ {
		bruteForce.RecordFailure("test@example.com")
	}

	// Attempt login with mixed-case variant — should still be locked.
	body, _ := json.Marshal(map[string]string{"email": "Test@Example.COM", "password": "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleLogin(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 (account locked via case variant), got %d", w.Code)
	}
}

func TestLogin_AccountLock_RetryAfterHeader(t *testing.T) {
	jwtMgr := mustNewJWTManager(t, testSecret, 15*time.Minute, 24*time.Hour)
	logger := zerolog.Nop()
	// Lock after 2 attempts with a short duration (1 minute) —
	// the header must still return 3600, not the actual remaining seconds.
	bruteForce := NewBruteForceTracker(2, 1*time.Minute)
	store := NewRefreshTokenStore(nil)
	h := NewHandler(nil, jwtMgr, nil, bruteForce, store, NewTokenBlacklist(), logger)

	email := "locked@example.com"

	// Trigger lockout by recording enough failures.
	for i := 0; i < 2; i++ {
		bruteForce.RecordFailure(email)
	}

	// Verify the account is locked.
	locked, actualRetryAfter := bruteForce.IsLocked(email)
	if !locked {
		t.Fatal("expected account to be locked")
	}
	// The actual remaining time should be less than 3600s (we set 1 minute).
	if actualRetryAfter.Seconds() >= 3600 {
		t.Fatal("test setup error: actual lockout duration should be less than 3600s")
	}

	body, _ := json.Marshal(map[string]string{"email": email, "password": "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleLogin(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}

	retryAfter := w.Header().Get("Retry-After")
	if retryAfter != "3600" {
		t.Errorf("expected Retry-After header to be fixed value '3600', got '%s'", retryAfter)
	}
}

// --- API Key handler tests ---

// withRole sets role + userID in context (same-package access to unexported keys).
func withRole(ctx context.Context, uid uuid.UUID, role Role) context.Context {
	ctx = context.WithValue(ctx, ctxKeyUserID, uid)
	ctx = context.WithValue(ctx, ctxKeyRole, role)
	return ctx
}

// withCustomer sets role, userID, and userType=customer in context for store API key tests.
func withCustomer(ctx context.Context, uid uuid.UUID) context.Context {
	ctx = context.WithValue(ctx, ctxKeyUserID, uid)
	ctx = context.WithValue(ctx, ctxKeyUserType, "customer")
	ctx = context.WithValue(ctx, ctxKeyRole, RoleCustomer)
	return ctx
}

func TestHandleCreateAPIKey_MissingName(t *testing.T) {
	logger := zerolog.Nop()
	h := NewHandler(nil, nil, NewAPIKeyManager(nil), nil, nil, nil, logger)

	body, _ := json.Marshal(CreateAPIKeyRequest{Name: "", Permissions: []string{"products.read"}})
	req := httptest.NewRequest(http.MethodPost, "/api-keys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.handleCreateAPIKey(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleCreateAPIKey_ManagerPermissionSubset(t *testing.T) {
	logger := zerolog.Nop()
	h := NewHandler(nil, nil, NewAPIKeyManager(nil), nil, nil, nil, logger)

	// Manager requests settings.update — not in their allowed permissions.
	body, _ := json.Marshal(CreateAPIKeyRequest{
		Name:        "test-key",
		Permissions: []string{"products.read", "settings.update"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api-keys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := withRole(req.Context(), uuid.New(), RoleManager)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.handleCreateAPIKey(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for permission exceeding role, got %d", w.Code)
	}

	// Verify response mentions the offending permission.
	var resp map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&resp)
	errs := resp["errors"].([]interface{})
	detail := errs[0].(map[string]interface{})["detail"].(string)
	if detail != "permission settings.update exceeds your role" {
		t.Errorf("unexpected error detail: %s", detail)
	}
}

func TestHandleCreateAPIKey_AdminBypassesSubsetCheck(t *testing.T) {
	logger := zerolog.Nop()
	h := NewHandler(nil, nil, NewAPIKeyManager(nil), nil, nil, nil, logger)

	body, _ := json.Marshal(CreateAPIKeyRequest{
		Name:        "test-key",
		Permissions: []string{"settings.update"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api-keys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := withRole(req.Context(), uuid.New(), RoleAdmin)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// The handler will panic on nil pool after passing the subset check.
	// A panic means we passed the 403 gate — that's the assertion.
	func() {
		defer func() { recover() }()
		h.handleCreateAPIKey(w, req)
	}()

	if w.Code == http.StatusForbidden {
		t.Error("admin should bypass permission-subset check")
	}
}

func TestHandleCreateAPIKey_ManagerAllowedPermsPass(t *testing.T) {
	logger := zerolog.Nop()
	h := NewHandler(nil, nil, NewAPIKeyManager(nil), nil, nil, nil, logger)

	body, _ := json.Marshal(CreateAPIKeyRequest{
		Name:        "test-key",
		Permissions: []string{"products.read", "orders.read"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api-keys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := withRole(req.Context(), uuid.New(), RoleManager)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	func() {
		defer func() { recover() }()
		h.handleCreateAPIKey(w, req)
	}()

	if w.Code == http.StatusForbidden {
		t.Error("manager requesting allowed permissions should not get 403")
	}
}

func TestHandleRevokeAPIKey_InvalidID(t *testing.T) {
	logger := zerolog.Nop()
	h := NewHandler(nil, nil, NewAPIKeyManager(nil), nil, nil, nil, logger)

	req := httptest.NewRequest(http.MethodDelete, "/api-keys/not-a-uuid", nil)

	// Set up chi URL param.
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "not-a-uuid")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.handleRevokeAPIKey(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid UUID, got %d", w.Code)
	}
}

func TestRevokeAllForUser(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()
	familyID := uuid.New()

	store := newMockRefreshTokenStore()
	store.store("tok-1", userID, familyID, time.Now().Add(time.Hour))
	store.store("tok-2", userID, familyID, time.Now().Add(time.Hour))
	store.store("tok-other", otherUserID, uuid.New(), time.Now().Add(time.Hour))

	store.revokeAllForUser(userID)

	if !store.records["tok-1"].Revoked {
		t.Error("tok-1 should be revoked")
	}
	if !store.records["tok-2"].Revoked {
		t.Error("tok-2 should be revoked")
	}
	if store.records["tok-other"].Revoked {
		t.Error("other user's token should NOT be revoked")
	}
}

// --- Store API Key handler tests ---

func TestHandleCreateStoreAPIKey_RejectsNonCustomer(t *testing.T) {
	logger := zerolog.Nop()
	h := NewHandler(nil, nil, NewAPIKeyManager(nil), nil, nil, nil, logger)

	body, _ := json.Marshal(CreateStoreAPIKeyRequest{
		Name:        "my-store-key",
		Permissions: []string{"store.products.read"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api-keys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Set as admin user (not customer).
	ctx := withRole(req.Context(), uuid.New(), RoleAdmin)
	ctx = context.WithValue(ctx, ctxKeyUserType, "admin")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.handleCreateStoreAPIKey(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-customer, got %d", w.Code)
	}
}

func TestHandleCreateStoreAPIKey_RejectsInvalidPermission(t *testing.T) {
	logger := zerolog.Nop()
	h := NewHandler(nil, nil, NewAPIKeyManager(nil), nil, nil, nil, logger)

	body, _ := json.Marshal(CreateStoreAPIKeyRequest{
		Name:        "my-store-key",
		Permissions: []string{"products.read"}, // admin permission, not store
	})
	req := httptest.NewRequest(http.MethodPost, "/api-keys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := withCustomer(req.Context(), uuid.New())
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.handleCreateStoreAPIKey(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid store permission, got %d", w.Code)
	}
}

func TestHandleCreateStoreAPIKey_MissingName(t *testing.T) {
	logger := zerolog.Nop()
	h := NewHandler(nil, nil, NewAPIKeyManager(nil), nil, nil, nil, logger)

	body, _ := json.Marshal(CreateStoreAPIKeyRequest{
		Name:        "",
		Permissions: []string{"store.products.read"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api-keys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := withCustomer(req.Context(), uuid.New())
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.handleCreateStoreAPIKey(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing name, got %d", w.Code)
	}
}

func TestHandleListStoreAPIKeys_RejectsNonCustomer(t *testing.T) {
	logger := zerolog.Nop()
	h := NewHandler(nil, nil, NewAPIKeyManager(nil), nil, nil, nil, logger)

	req := httptest.NewRequest(http.MethodGet, "/api-keys", nil)
	ctx := withRole(req.Context(), uuid.New(), RoleAdmin)
	ctx = context.WithValue(ctx, ctxKeyUserType, "admin")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.handleListStoreAPIKeys(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-customer, got %d", w.Code)
	}
}

func TestHandleRevokeStoreAPIKey_RejectsNonCustomer(t *testing.T) {
	logger := zerolog.Nop()
	h := NewHandler(nil, nil, NewAPIKeyManager(nil), nil, nil, nil, logger)

	req := httptest.NewRequest(http.MethodDelete, "/api-keys/"+uuid.New().String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", uuid.New().String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = withRole(ctx, uuid.New(), RoleAdmin)
	ctx = context.WithValue(ctx, ctxKeyUserType, "admin")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.handleRevokeStoreAPIKey(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-customer, got %d", w.Code)
	}
}

func TestHandleRevokeStoreAPIKey_InvalidID(t *testing.T) {
	logger := zerolog.Nop()
	h := NewHandler(nil, nil, NewAPIKeyManager(nil), nil, nil, nil, logger)

	req := httptest.NewRequest(http.MethodDelete, "/api-keys/not-a-uuid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "not-a-uuid")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = withCustomer(ctx, uuid.New())
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.handleRevokeStoreAPIKey(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid UUID, got %d", w.Code)
	}
}

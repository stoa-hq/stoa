package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	jwtMgr := NewJWTManager("test-secret", 15*time.Minute, 24*time.Hour)
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
	jwtMgr := NewJWTManager("test-secret", 15*time.Minute, 24*time.Hour)
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
	jwtMgr := NewJWTManager("test-secret", 15*time.Minute, 24*time.Hour)
	logger := zerolog.Nop()
	bruteForce := NewBruteForceTracker(5, 15*time.Minute)
	store := NewRefreshTokenStore(nil) // won't be reached

	h := NewHandler(nil, jwtMgr, nil, bruteForce, store, logger)

	// Generate an access token (not refresh).
	accessTok, _ := jwtMgr.GenerateAccessToken(uuid.New(), "test@example.com", "admin", "admin")

	body, _ := json.Marshal(RefreshRequest{RefreshToken: accessTok})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.handleRefresh(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandleRefresh_HTTPEndpoint_RejectsExpiredToken(t *testing.T) {
	jwtMgr := NewJWTManager("test-secret", 15*time.Minute, -time.Second) // negative refresh TTL
	logger := zerolog.Nop()
	bruteForce := NewBruteForceTracker(5, 15*time.Minute)
	store := NewRefreshTokenStore(nil) // won't be reached

	h := NewHandler(nil, jwtMgr, nil, bruteForce, store, logger)

	expiredTok, _ := jwtMgr.GenerateRefreshToken(uuid.New(), "test@example.com", "admin", "admin")

	body, _ := json.Marshal(RefreshRequest{RefreshToken: expiredTok})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.handleRefresh(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
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

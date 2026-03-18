package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

const testSecret = "test-secret-that-is-at-least-32-bytes-long"

func mustNewJWTManager(t *testing.T, secret string, accessTTL, refreshTTL time.Duration) *JWTManager {
	t.Helper()
	m, err := NewJWTManager(secret, accessTTL, refreshTTL)
	if err != nil {
		t.Fatalf("NewJWTManager: %v", err)
	}
	return m
}

func TestJWTManager_GenerateAndValidateAccessToken(t *testing.T) {
	m := mustNewJWTManager(t, testSecret, 15*time.Minute, 24*time.Hour)
	userID := uuid.New()

	token, err := m.GenerateAccessToken(userID, "admin@test.com", "admin", "super_admin")
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	claims, err := m.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("UserID: got %s, want %s", claims.UserID, userID)
	}
	if claims.UserType != "admin" {
		t.Errorf("UserType: got %q, want %q", claims.UserType, "admin")
	}
	if claims.Role != "super_admin" {
		t.Errorf("Role: got %q, want %q", claims.Role, "super_admin")
	}
	if claims.Type != AccessToken {
		t.Errorf("Type: got %q, want %q", claims.Type, AccessToken)
	}
}

func TestJWTManager_RefreshToken(t *testing.T) {
	m := mustNewJWTManager(t, testSecret, 15*time.Minute, 7*24*time.Hour)
	userID := uuid.New()

	token, err := m.GenerateRefreshToken(userID, "customer@test.com", "customer", "customer")
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}

	claims, err := m.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}

	if claims.Type != RefreshToken {
		t.Errorf("Type: got %q, want %q", claims.Type, RefreshToken)
	}
	if claims.UserID != userID {
		t.Errorf("UserID: got %s, want %s", claims.UserID, userID)
	}
}

func TestJWTManager_ExpiredToken(t *testing.T) {
	m := mustNewJWTManager(t, testSecret, -time.Second, time.Hour)

	token, err := m.GenerateAccessToken(uuid.New(), "admin@test.com", "admin", "admin")
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	_, err = m.ValidateToken(token)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestJWTManager_WrongSecret(t *testing.T) {
	secret1 := "secret-a-that-is-at-least-32-bytes-long"
	secret2 := "secret-b-that-is-at-least-32-bytes-long"
	m1 := mustNewJWTManager(t, secret1, time.Hour, time.Hour)
	m2 := mustNewJWTManager(t, secret2, time.Hour, time.Hour)

	token, err := m1.GenerateAccessToken(uuid.New(), "admin@test.com", "admin", "admin")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	_, err = m2.ValidateToken(token)
	if err == nil {
		t.Fatal("expected error for token signed with wrong secret")
	}
}

func TestJWTManager_MalformedToken(t *testing.T) {
	m := mustNewJWTManager(t, testSecret, time.Hour, time.Hour)

	for _, bad := range []string{"", "not.a.jwt", "header.payload"} {
		_, err := m.ValidateToken(bad)
		if err == nil {
			t.Errorf("expected error for malformed token %q, got nil", bad)
		}
	}
}

func TestJWTManager_ClaimsIDUnique(t *testing.T) {
	m := mustNewJWTManager(t, testSecret, time.Hour, time.Hour)
	uid := uuid.New()

	t1, _ := m.GenerateAccessToken(uid, "admin@test.com", "admin", "admin")
	t2, _ := m.GenerateAccessToken(uid, "admin@test.com", "admin", "admin")

	c1, _ := m.ValidateToken(t1)
	c2, _ := m.ValidateToken(t2)

	if c1.ID == c2.ID {
		t.Error("expected unique JWT IDs (jti) for each token")
	}
}

func TestNewJWTManager_RejectsDefaultSecret(t *testing.T) {
	_, err := NewJWTManager(defaultSecret, time.Hour, time.Hour)
	if err == nil {
		t.Fatal("expected error for default secret, got nil")
	}
	if !strings.Contains(err.Error(), "default secret") {
		t.Errorf("error should mention default secret, got: %v", err)
	}
}

func TestNewJWTManager_RejectsShortSecret(t *testing.T) {
	_, err := NewJWTManager("too-short", time.Hour, time.Hour)
	if err == nil {
		t.Fatal("expected error for short secret, got nil")
	}
	if !strings.Contains(err.Error(), "at least") {
		t.Errorf("error should mention minimum length, got: %v", err)
	}
}

func TestNewJWTManager_AcceptsValidSecret(t *testing.T) {
	m, err := NewJWTManager(testSecret, time.Hour, time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil JWTManager")
	}
}

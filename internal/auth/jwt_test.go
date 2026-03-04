package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWTManager_GenerateAndValidateAccessToken(t *testing.T) {
	m := NewJWTManager("test-secret", 15*time.Minute, 24*time.Hour)
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
	m := NewJWTManager("secret", 15*time.Minute, 7*24*time.Hour)
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
	// Negative TTL → immediately expired token.
	m := NewJWTManager("secret", -time.Second, time.Hour)

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
	m1 := NewJWTManager("secret-a", time.Hour, time.Hour)
	m2 := NewJWTManager("secret-b", time.Hour, time.Hour)

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
	m := NewJWTManager("secret", time.Hour, time.Hour)

	for _, bad := range []string{"", "not.a.jwt", "header.payload"} {
		_, err := m.ValidateToken(bad)
		if err == nil {
			t.Errorf("expected error for malformed token %q, got nil", bad)
		}
	}
}

func TestJWTManager_ClaimsIDUnique(t *testing.T) {
	m := NewJWTManager("secret", time.Hour, time.Hour)
	uid := uuid.New()

	t1, _ := m.GenerateAccessToken(uid, "admin@test.com", "admin", "admin")
	t2, _ := m.GenerateAccessToken(uid, "admin@test.com", "admin", "admin")

	c1, _ := m.ValidateToken(t1)
	c2, _ := m.ValidateToken(t2)

	if c1.ID == c2.ID {
		t.Error("expected unique JWT IDs (jti) for each token")
	}
}

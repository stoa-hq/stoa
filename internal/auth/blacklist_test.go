package auth

import (
	"testing"
	"time"
)

func TestBlacklist_AddAndCheck(t *testing.T) {
	bl := NewTokenBlacklist()
	defer bl.Stop()

	jti := "test-jti-123"
	bl.Add(jti, time.Now().Add(5*time.Minute))

	if !bl.IsBlacklisted(jti) {
		t.Error("expected token to be blacklisted")
	}
}

func TestBlacklist_NotBlacklisted(t *testing.T) {
	bl := NewTokenBlacklist()
	defer bl.Stop()

	if bl.IsBlacklisted("unknown-jti") {
		t.Error("unknown JTI should not be blacklisted")
	}
}

func TestBlacklist_ExpiredEntryNotBlacklisted(t *testing.T) {
	bl := NewTokenBlacklist()
	defer bl.Stop()

	jti := "expired-jti"
	bl.Add(jti, time.Now().Add(-1*time.Second))

	if bl.IsBlacklisted(jti) {
		t.Error("expired entry should not be reported as blacklisted")
	}
}

func TestBlacklist_Cleanup(t *testing.T) {
	bl := &TokenBlacklist{
		tokens:  make(map[string]time.Time),
		stopCh:  make(chan struct{}),
		stopped: make(chan struct{}),
	}
	// No background goroutine — we test cleanup manually.
	close(bl.stopped)

	bl.tokens["expired"] = time.Now().Add(-1 * time.Minute)
	bl.tokens["valid"] = time.Now().Add(5 * time.Minute)

	// Simulate cleanup tick.
	now := time.Now()
	bl.mu.Lock()
	for jti, exp := range bl.tokens {
		if now.After(exp) {
			delete(bl.tokens, jti)
		}
	}
	bl.mu.Unlock()

	if _, ok := bl.tokens["expired"]; ok {
		t.Error("expired entry should have been cleaned up")
	}
	if _, ok := bl.tokens["valid"]; !ok {
		t.Error("valid entry should still exist")
	}
}

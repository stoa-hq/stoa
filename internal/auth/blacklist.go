package auth

import (
	"sync"
	"time"
)

// TokenBlacklist is an in-memory blacklist for revoked JWT access tokens.
// Entries are keyed by JTI and expire automatically when the token's TTL
// elapses, so no external store (Redis, DB) is needed.
type TokenBlacklist struct {
	mu      sync.RWMutex
	tokens  map[string]time.Time // JTI → expiry
	stopCh  chan struct{}
	stopped chan struct{}
}

// NewTokenBlacklist creates a blacklist and starts a background goroutine
// that purges expired entries every minute.
func NewTokenBlacklist() *TokenBlacklist {
	bl := &TokenBlacklist{
		tokens:  make(map[string]time.Time),
		stopCh:  make(chan struct{}),
		stopped: make(chan struct{}),
	}
	go bl.cleanup()
	return bl
}

// Add blacklists a token identified by its JTI until expiresAt.
func (bl *TokenBlacklist) Add(jti string, expiresAt time.Time) {
	bl.mu.Lock()
	bl.tokens[jti] = expiresAt
	bl.mu.Unlock()
}

// IsBlacklisted returns true if the JTI is in the blacklist and has not yet expired.
func (bl *TokenBlacklist) IsBlacklisted(jti string) bool {
	bl.mu.RLock()
	exp, ok := bl.tokens[jti]
	bl.mu.RUnlock()
	if !ok {
		return false
	}
	return time.Now().Before(exp)
}

// Stop terminates the background cleanup goroutine.
func (bl *TokenBlacklist) Stop() {
	close(bl.stopCh)
	<-bl.stopped
}

func (bl *TokenBlacklist) cleanup() {
	defer close(bl.stopped)
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-bl.stopCh:
			return
		case now := <-ticker.C:
			bl.mu.Lock()
			for jti, exp := range bl.tokens {
				if now.After(exp) {
					delete(bl.tokens, jti)
				}
			}
			bl.mu.Unlock()
		}
	}
}

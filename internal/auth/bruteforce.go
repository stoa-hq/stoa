package auth

import (
	"sync"
	"time"
)

type loginAttempt struct {
	Count    int
	LastFail time.Time
	LockedAt time.Time
}

type BruteForceTracker struct {
	mu           sync.Mutex
	attempts     map[string]*loginAttempt
	maxAttempts  int
	lockDuration time.Duration
	stopCleanup  chan struct{}
}

func NewBruteForceTracker(maxAttempts int, lockDuration time.Duration) *BruteForceTracker {
	t := &BruteForceTracker{
		attempts:     make(map[string]*loginAttempt),
		maxAttempts:  maxAttempts,
		lockDuration: lockDuration,
		stopCleanup:  make(chan struct{}),
	}
	go t.cleanupLoop()
	return t
}

func (t *BruteForceTracker) IsLocked(email string) (bool, time.Duration) {
	key := NormalizeEmail(email)
	t.mu.Lock()
	defer t.mu.Unlock()

	a, ok := t.attempts[key]
	if !ok {
		return false, 0
	}

	if a.LockedAt.IsZero() {
		return false, 0
	}

	remaining := t.lockDuration - time.Since(a.LockedAt)
	if remaining <= 0 {
		delete(t.attempts, key)
		return false, 0
	}

	return true, remaining
}

func (t *BruteForceTracker) RecordFailure(email string) {
	key := NormalizeEmail(email)
	t.mu.Lock()
	defer t.mu.Unlock()

	a, ok := t.attempts[key]
	if !ok {
		a = &loginAttempt{}
		t.attempts[key] = a
	}

	a.Count++
	a.LastFail = time.Now()

	if a.Count >= t.maxAttempts {
		a.LockedAt = time.Now()
	}
}

func (t *BruteForceTracker) RecordSuccess(email string) {
	key := NormalizeEmail(email)
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.attempts, key)
}

func (t *BruteForceTracker) Stop() {
	close(t.stopCleanup)
}

func (t *BruteForceTracker) cleanupLoop() {
	ticker := time.NewTicker(t.lockDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.mu.Lock()
			now := time.Now()
			for key, a := range t.attempts {
				if !a.LockedAt.IsZero() && now.Sub(a.LockedAt) >= t.lockDuration {
					delete(t.attempts, key)
				} else if a.LockedAt.IsZero() && now.Sub(a.LastFail) >= t.lockDuration {
					delete(t.attempts, key)
				}
			}
			t.mu.Unlock()
		case <-t.stopCleanup:
			return
		}
	}
}

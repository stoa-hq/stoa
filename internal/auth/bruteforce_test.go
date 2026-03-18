package auth

import (
	"sync"
	"testing"
	"time"
)

func TestBruteForce_UnderThreshold(t *testing.T) {
	tracker := NewBruteForceTracker(5, time.Minute)
	defer tracker.Stop()

	for i := 0; i < 4; i++ {
		tracker.RecordFailure("user@example.com")
	}

	locked, _ := tracker.IsLocked("user@example.com")
	if locked {
		t.Fatal("expected account to not be locked under threshold")
	}
}

func TestBruteForce_LocksAfterMaxAttempts(t *testing.T) {
	tracker := NewBruteForceTracker(3, time.Minute)
	defer tracker.Stop()

	for i := 0; i < 3; i++ {
		tracker.RecordFailure("user@example.com")
	}

	locked, retryAfter := tracker.IsLocked("user@example.com")
	if !locked {
		t.Fatal("expected account to be locked after max attempts")
	}
	if retryAfter <= 0 {
		t.Fatalf("expected positive retry-after, got %v", retryAfter)
	}
}

func TestBruteForce_ResetOnSuccess(t *testing.T) {
	tracker := NewBruteForceTracker(3, time.Minute)
	defer tracker.Stop()

	tracker.RecordFailure("user@example.com")
	tracker.RecordFailure("user@example.com")
	tracker.RecordSuccess("user@example.com")

	// After success, counter should be reset — 3 more failures needed to lock
	tracker.RecordFailure("user@example.com")
	tracker.RecordFailure("user@example.com")

	locked, _ := tracker.IsLocked("user@example.com")
	if locked {
		t.Fatal("expected account to not be locked after success reset")
	}
}

func TestBruteForce_UnlocksAfterDuration(t *testing.T) {
	tracker := NewBruteForceTracker(2, 50*time.Millisecond)
	defer tracker.Stop()

	tracker.RecordFailure("user@example.com")
	tracker.RecordFailure("user@example.com")

	locked, _ := tracker.IsLocked("user@example.com")
	if !locked {
		t.Fatal("expected account to be locked")
	}

	time.Sleep(60 * time.Millisecond)

	locked, _ = tracker.IsLocked("user@example.com")
	if locked {
		t.Fatal("expected account to be unlocked after duration")
	}
}

func TestBruteForce_EmailNormalization(t *testing.T) {
	tracker := NewBruteForceTracker(3, time.Minute)
	defer tracker.Stop()

	tracker.RecordFailure("User@Example.COM")
	tracker.RecordFailure("user@example.com")
	tracker.RecordFailure("USER@EXAMPLE.COM")

	locked, _ := tracker.IsLocked("user@example.com")
	if !locked {
		t.Fatal("expected case-insensitive matching to lock account")
	}
}

func TestBruteForce_IndependentEmails(t *testing.T) {
	tracker := NewBruteForceTracker(2, time.Minute)
	defer tracker.Stop()

	tracker.RecordFailure("alice@example.com")
	tracker.RecordFailure("alice@example.com")

	tracker.RecordFailure("bob@example.com")

	lockedAlice, _ := tracker.IsLocked("alice@example.com")
	lockedBob, _ := tracker.IsLocked("bob@example.com")

	if !lockedAlice {
		t.Fatal("expected alice to be locked")
	}
	if lockedBob {
		t.Fatal("expected bob to not be locked")
	}
}

func TestBruteForce_ConcurrentAccess(t *testing.T) {
	tracker := NewBruteForceTracker(100, time.Minute)
	defer tracker.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tracker.RecordFailure("concurrent@example.com")
			tracker.IsLocked("concurrent@example.com")
		}()
	}
	wg.Wait()

	locked, _ := tracker.IsLocked("concurrent@example.com")
	if !locked {
		t.Fatal("expected account to be locked after 200 concurrent failures with threshold 100")
	}
}

func TestBruteForce_CleanupExpiredEntries(t *testing.T) {
	tracker := NewBruteForceTracker(2, 50*time.Millisecond)
	defer tracker.Stop()

	tracker.RecordFailure("cleanup@example.com")
	tracker.RecordFailure("cleanup@example.com")

	// Wait for lock to expire and cleanup to run
	time.Sleep(120 * time.Millisecond)

	tracker.mu.Lock()
	_, exists := tracker.attempts["cleanup@example.com"]
	tracker.mu.Unlock()

	if exists {
		t.Fatal("expected expired entry to be cleaned up")
	}
}

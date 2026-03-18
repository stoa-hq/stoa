package auth

import (
	"math"
	"testing"
	"time"
)

func TestHashAndVerify(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	match, err := VerifyPassword("correct-horse-battery-staple", hash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !match {
		t.Error("expected password to match its hash")
	}
}

func TestVerify_WrongPassword(t *testing.T) {
	hash, err := HashPassword("correct")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	match, err := VerifyPassword("wrong", hash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if match {
		t.Error("expected mismatch for wrong password")
	}
}

func TestHash_UniqueSalts(t *testing.T) {
	// Two hashes of the same password must differ (random salt per call).
	h1, err1 := HashPassword("same-input")
	h2, err2 := HashPassword("same-input")
	if err1 != nil || err2 != nil {
		t.Fatalf("HashPassword errors: %v / %v", err1, err2)
	}
	if h1 == h2 {
		t.Error("expected unique hashes for the same password (random salt)")
	}

	// Both hashes must still verify correctly.
	ok1, _ := VerifyPassword("same-input", h1)
	ok2, _ := VerifyPassword("same-input", h2)
	if !ok1 || !ok2 {
		t.Error("both salted hashes should verify correctly")
	}
}

func TestVerify_CorruptHash(t *testing.T) {
	for _, bad := range []string{"", "not-a-hash", "$argon2id$wrong"} {
		_, err := VerifyPassword("any", bad)
		if err == nil {
			t.Errorf("expected error for corrupt hash %q, got nil", bad)
		}
	}
}

func TestDummyHash_Initialized(t *testing.T) {
	if dummyHash == "" {
		t.Fatal("dummyHash should be initialized at package init")
	}

	// dummyHash must be a valid Argon2id hash that VerifyPassword can decode
	_, err := VerifyPassword("any-password", dummyHash)
	if err != nil {
		t.Fatalf("dummyHash should be a valid hash, got error: %v", err)
	}
}

func TestDummyHash_TimingSafe(t *testing.T) {
	// Verify that VerifyPassword against a real hash and against the dummyHash
	// take approximately the same time, preventing timing-based user enumeration.
	realHash, err := HashPassword("real-user-password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	const iterations = 5

	// Measure real hash verification
	var realTotal time.Duration
	for i := 0; i < iterations; i++ {
		start := time.Now()
		VerifyPassword("wrong-password", realHash)
		realTotal += time.Since(start)
	}
	realAvg := realTotal / time.Duration(iterations)

	// Measure dummy hash verification
	var dummyTotal time.Duration
	for i := 0; i < iterations; i++ {
		start := time.Now()
		VerifyPassword("wrong-password", dummyHash)
		dummyTotal += time.Since(start)
	}
	dummyAvg := dummyTotal / time.Duration(iterations)

	diff := math.Abs(float64(realAvg - dummyAvg))
	threshold := 50 * time.Millisecond

	if diff > float64(threshold) {
		t.Errorf("timing difference between real and dummy hash too large: real=%v, dummy=%v, diff=%v (threshold=%v)",
			realAvg, dummyAvg, time.Duration(diff), threshold)
	}
}

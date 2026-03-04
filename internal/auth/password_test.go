package auth

import "testing"

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

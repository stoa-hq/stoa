package crypto

import (
	"crypto/rand"
	"testing"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	return key
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := testKey(t)
	plaintext := []byte(`{"api_key":"sk_live_abc123"}`)

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if string(ciphertext) == string(plaintext) {
		t.Fatal("ciphertext should differ from plaintext")
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptWrongKeyLength(t *testing.T) {
	_, err := Encrypt([]byte("hello"), []byte("short"))
	if err == nil {
		t.Fatal("expected error for short key")
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	key := testKey(t)
	plaintext := []byte(`{"secret":"value"}`)

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Tamper with the ciphertext (flip a byte after the nonce)
	ciphertext[nonceSize+1] ^= 0xff

	_, err = Decrypt(ciphertext, key)
	if err == nil {
		t.Fatal("expected error for tampered ciphertext")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key1 := testKey(t)
	key2 := testKey(t)

	ciphertext, err := Encrypt([]byte("secret"), key1)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	_, err = Decrypt(ciphertext, key2)
	if err == nil {
		t.Fatal("expected error for wrong key")
	}
}

func TestDecryptTooShort(t *testing.T) {
	key := testKey(t)
	_, err := Decrypt([]byte("short"), key)
	if err == nil {
		t.Fatal("expected error for short ciphertext")
	}
}

func TestIsEncrypted(t *testing.T) {
	key := testKey(t)

	plainJSON := []byte(`{"api_key":"sk_test_123"}`)
	if IsEncrypted(plainJSON) {
		t.Error("plain JSON should not be detected as encrypted")
	}

	encrypted, err := Encrypt(plainJSON, key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if !IsEncrypted(encrypted) {
		t.Error("encrypted data should be detected as encrypted")
	}

	// Too short data
	if IsEncrypted([]byte("ab")) {
		t.Error("short data should not be detected as encrypted")
	}

	// Empty / nil
	if IsEncrypted(nil) {
		t.Error("nil should not be detected as encrypted")
	}
}

func TestParseKeyRaw32(t *testing.T) {
	raw := "abcdefghijklmnopqrstuvwxyz012345"
	key, err := ParseKey(raw)
	if err != nil {
		t.Fatalf("ParseKey: %v", err)
	}
	if string(key) != raw {
		t.Fatalf("got %q, want %q", key, raw)
	}
}

func TestParseKeyHex64(t *testing.T) {
	hex64 := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	key, err := ParseKey(hex64)
	if err != nil {
		t.Fatalf("ParseKey: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(key))
	}
}

func TestParseKeyInvalidLength(t *testing.T) {
	_, err := ParseKey("too-short")
	if err == nil {
		t.Fatal("expected error for invalid key length")
	}
}

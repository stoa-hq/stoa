package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const nonceSize = 12

// ParseKey accepts a 32-byte raw key or a 64-character hex-encoded key and
// returns the 32-byte key suitable for AES-256.
func ParseKey(key string) ([]byte, error) {
	if len(key) == 64 {
		b, err := hex.DecodeString(key)
		if err != nil {
			return nil, fmt.Errorf("crypto: invalid hex key: %w", err)
		}
		return b, nil
	}
	if len(key) == 32 {
		return []byte(key), nil
	}
	return nil, errors.New("crypto: encryption key must be 32 bytes or 64 hex characters")
}

// Encrypt encrypts plaintext using AES-256-GCM with a random 12-byte nonce.
// The output format is nonce || ciphertext (nonce prepended).
func Encrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: new gcm: %w", err)
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("crypto: generating nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts data produced by Encrypt (nonce || ciphertext).
func Decrypt(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: new gcm: %w", err)
	}

	if len(data) < nonceSize+gcm.Overhead() {
		return nil, errors.New("crypto: ciphertext too short")
	}

	nonce := data[:nonceSize]
	ciphertext := data[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: decryption failed: %w", err)
	}
	return plaintext, nil
}

// IsEncrypted returns true if data looks like an encrypted blob rather than
// plain JSON. It checks that the data is long enough to contain a nonce +
// GCM tag and is not valid JSON.
func IsEncrypted(data []byte) bool {
	// GCM overhead = 16 byte tag; minimum encrypted length = nonce + tag
	if len(data) < nonceSize+16 {
		return false
	}
	return !json.Valid(data)
}

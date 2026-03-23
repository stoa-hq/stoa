package sdk

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestGenerateNonce(t *testing.T) {
	nonce := GenerateNonce()

	decoded, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		t.Fatalf("nonce is not valid base64: %v", err)
	}
	if len(decoded) != 16 {
		t.Errorf("nonce decoded length = %d, want 16", len(decoded))
	}

	nonce2 := GenerateNonce()
	if nonce == nonce2 {
		t.Error("two consecutive nonces are identical")
	}
}

func TestInjectNonce(t *testing.T) {
	html := []byte(`<html><head><style>body{}</style><script src="/app.js"></script></head><body><script>x()</script></body></html>`)
	result := InjectNonce(html, "n123")
	resultStr := string(result)

	if got := strings.Count(resultStr, `<script nonce="n123"`); got != 2 {
		t.Errorf("expected 2 script nonces, got %d", got)
	}
	if got := strings.Count(resultStr, `<style nonce="n123"`); got != 1 {
		t.Errorf("expected 1 style nonce, got %d", got)
	}
}

func TestInjectNonceNoTags(t *testing.T) {
	html := []byte(`<html><body>hello</body></html>`)
	result := InjectNonce(html, "n123")
	if string(result) != string(html) {
		t.Error("InjectNonce modified HTML without script/style tags")
	}
}

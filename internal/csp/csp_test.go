package csp

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestGenerateNonce(t *testing.T) {
	nonce := GenerateNonce()

	// Must be valid base64.
	decoded, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		t.Fatalf("nonce is not valid base64: %v", err)
	}

	// Must be 16 bytes (128 bits).
	if len(decoded) != 16 {
		t.Errorf("nonce decoded length = %d, want 16", len(decoded))
	}

	// Two nonces must differ.
	nonce2 := GenerateNonce()
	if nonce == nonce2 {
		t.Error("two consecutive nonces are identical")
	}
}

func TestApply(t *testing.T) {
	template := "script-src 'self' 'nonce-{{NONCE}}' 'strict-dynamic'; style-src 'self' 'nonce-{{NONCE}}'"
	result := Apply(template, "abc123")

	if strings.Contains(result, "{{NONCE}}") {
		t.Error("result still contains {{NONCE}} placeholder")
	}

	want := "script-src 'self' 'nonce-abc123' 'strict-dynamic'; style-src 'self' 'nonce-abc123'"
	if result != want {
		t.Errorf("Apply() = %q, want %q", result, want)
	}
}

func TestApplyNoPlaceholder(t *testing.T) {
	template := "default-src 'self'"
	result := Apply(template, "abc123")
	if result != template {
		t.Errorf("Apply() modified template without placeholder: %q", result)
	}
}

func TestInjectNonce(t *testing.T) {
	html := []byte(`<!DOCTYPE html><html><head><style>body{}</style><script type="module" src="/app.js"></script></head><body><script>console.log("hi")</script></body></html>`)
	result := InjectNonce(html, "test-nonce")

	resultStr := string(result)

	scriptCount := strings.Count(resultStr, `<script nonce="test-nonce"`)
	if scriptCount != 2 {
		t.Errorf("expected 2 script nonce attributes, got %d in: %s", scriptCount, resultStr)
	}

	styleCount := strings.Count(resultStr, `<style nonce="test-nonce"`)
	if styleCount != 1 {
		t.Errorf("expected 1 style nonce attribute, got %d in: %s", styleCount, resultStr)
	}

	if strings.Contains(resultStr, "<script type") {
		t.Error("original <script without nonce still present")
	}
	if strings.Contains(resultStr, "<style>") {
		t.Error("original <style without nonce still present")
	}
}

func TestInjectNonceNoTags(t *testing.T) {
	html := []byte(`<!DOCTYPE html><html><head></head><body></body></html>`)
	result := InjectNonce(html, "test-nonce")
	if string(result) != string(html) {
		t.Error("InjectNonce modified HTML without script/style tags")
	}
}

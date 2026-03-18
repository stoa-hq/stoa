// Package csp provides nonce-based Content-Security-Policy utilities.
// It generates per-request nonces and injects them into CSP headers
// and HTML script tags, replacing 'unsafe-inline' with 'strict-dynamic'.
package csp

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
)

const noncePlaceholder = "{{NONCE}}"

// GenerateNonce returns a cryptographically random 128-bit nonce
// encoded as base64.
func GenerateNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("csp: " + err.Error())
	}
	return base64.StdEncoding.EncodeToString(b)
}

// Apply replaces all {{NONCE}} placeholders in a CSP template with the given nonce.
func Apply(cspTemplate, nonce string) string {
	return strings.ReplaceAll(cspTemplate, noncePlaceholder, nonce)
}

// InjectNonce adds a nonce attribute to every <script tag in the HTML.
func InjectNonce(html []byte, nonce string) []byte {
	return []byte(strings.ReplaceAll(string(html), "<script", `<script nonce="`+nonce+`"`))
}

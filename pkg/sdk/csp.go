package sdk

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
)

// GenerateNonce returns a cryptographically random 128-bit nonce
// encoded as base64. Use this in plugin handlers that serve HTML pages
// to create nonce-based Content-Security-Policy headers.
func GenerateNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("csp: " + err.Error())
	}
	return base64.StdEncoding.EncodeToString(b)
}

// InjectNonce adds a nonce attribute to every <script> and <style> tag
// in the HTML. Use together with GenerateNonce to serve HTML pages with
// nonce-based CSP.
func InjectNonce(html []byte, nonce string) []byte {
	s := string(html)
	attr := ` nonce="` + nonce + `"`
	s = strings.ReplaceAll(s, "<script", "<script"+attr)
	s = strings.ReplaceAll(s, "<style", "<style"+attr)
	return []byte(s)
}

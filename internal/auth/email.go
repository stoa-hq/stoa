package auth

import "strings"

// NormalizeEmail lowercases and trims whitespace from an email address.
// This ensures consistent lookups across the brute-force tracker, database
// queries, and uniqueness checks.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

package auth

import "testing"

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"lowercase passthrough", "user@example.com", "user@example.com"},
		{"mixed case", "User@Example.COM", "user@example.com"},
		{"all uppercase", "ADMIN@STORE.IO", "admin@store.io"},
		{"leading whitespace", "  user@example.com", "user@example.com"},
		{"trailing whitespace", "user@example.com  ", "user@example.com"},
		{"both whitespace", "  User@Example.COM  ", "user@example.com"},
		{"empty string", "", ""},
		{"only whitespace", "   ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeEmail(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeEmail(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

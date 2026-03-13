package mcp

import (
	"strings"
	"testing"
)

func TestValidateStorePath_Valid(t *testing.T) {
	validPaths := []string{
		"/api/v1/store/products",
		"/api/v1/store/stripe/payment-intent",
		"/api/v1/store/cart/items",
	}
	for _, path := range validPaths {
		if err := validateStorePath(path); err != nil {
			t.Errorf("validateStorePath(%q) = %v, want nil", path, err)
		}
	}
}

func TestValidateStorePath_Rejects_AdminPaths(t *testing.T) {
	blockedPaths := []string{
		"/api/v1/admin/products",
		"/api/v1/admin/users",
		"/plugins/stripe/webhook",
		"/",
		"",
	}
	for _, path := range blockedPaths {
		err := validateStorePath(path)
		if err == nil {
			t.Errorf("validateStorePath(%q) = nil, want error", path)
			continue
		}
		if !strings.Contains(err.Error(), "access denied") {
			t.Errorf("validateStorePath(%q) error = %v, want 'access denied'", path, err)
		}
	}
}

func TestValidateStorePath_Rejects_PathTraversal(t *testing.T) {
	traversalPaths := []string{
		"/api/v1/store/../admin/users",
		"/api/v1/store/..%2fadmin",
	}
	for _, path := range traversalPaths {
		err := validateStorePath(path)
		if err == nil {
			t.Errorf("validateStorePath(%q) = nil, want error", path)
		}
	}
}

package auth

import (
	"context"
	"testing"
)

func TestAllStorePermissions_ReturnsAll6(t *testing.T) {
	perms := AllStorePermissions()
	if len(perms) != 6 {
		t.Fatalf("AllStorePermissions() returned %d permissions, want 6", len(perms))
	}

	expected := []Permission{
		PermStoreProductRead,
		PermStoreCartManage,
		PermStoreCheckout,
		PermStoreAccountRead,
		PermStoreAccountUpdate,
		PermStoreOrdersRead,
	}
	for i, want := range expected {
		if perms[i] != want {
			t.Errorf("AllStorePermissions()[%d] = %q, want %q", i, perms[i], want)
		}
	}
}

func TestIsStorePermission_StorePerms(t *testing.T) {
	for _, p := range AllStorePermissions() {
		if !IsStorePermission(p) {
			t.Errorf("IsStorePermission(%q) = false, want true", p)
		}
	}
}

func TestIsStorePermission_AdminPerms(t *testing.T) {
	adminPerms := []Permission{
		PermProductCreate,
		PermProductRead,
		PermOrderRead,
		PermSettingsUpdate,
		PermAuditRead,
		PermAPIKeysManage,
	}
	for _, p := range adminPerms {
		if IsStorePermission(p) {
			t.Errorf("IsStorePermission(%q) = true, want false", p)
		}
	}
}

func TestHasPermissionCtx_StoreKeyWithContextPerms(t *testing.T) {
	// Simulate a store API key: RoleCustomer with explicit context permissions.
	ctx := context.WithValue(context.Background(), ctxKeyPermissions, []Permission{
		PermStoreProductRead,
		PermStoreCartManage,
	})

	// Should find permissions that are in context.
	if !HasPermissionCtx(ctx, RoleCustomer, PermStoreProductRead) {
		t.Error("expected HasPermissionCtx to return true for PermStoreProductRead")
	}
	if !HasPermissionCtx(ctx, RoleCustomer, PermStoreCartManage) {
		t.Error("expected HasPermissionCtx to return true for PermStoreCartManage")
	}

	// Should deny permissions not in context.
	if HasPermissionCtx(ctx, RoleCustomer, PermStoreCheckout) {
		t.Error("expected HasPermissionCtx to return false for PermStoreCheckout (not in context)")
	}
}

func TestHasPermissionCtx_CustomerWithoutContextPerms(t *testing.T) {
	// Regular customer (JWT login, no context permissions) — falls back to role-based.
	ctx := context.Background()

	// RoleCustomer has empty role-based permissions, so everything should be denied.
	if HasPermissionCtx(ctx, RoleCustomer, PermStoreProductRead) {
		t.Error("expected HasPermissionCtx to return false for regular customer without context perms")
	}
}

func TestHasPermissionCtx_APIClientWithContextPerms(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxKeyPermissions, []Permission{
		PermProductRead,
	})

	if !HasPermissionCtx(ctx, RoleAPIClient, PermProductRead) {
		t.Error("expected HasPermissionCtx to return true for API client with PermProductRead")
	}
	if HasPermissionCtx(ctx, RoleAPIClient, PermProductCreate) {
		t.Error("expected HasPermissionCtx to return false for API client without PermProductCreate")
	}
}

func TestHasPermissionCtx_AdminRole(t *testing.T) {
	ctx := context.Background()

	if !HasPermissionCtx(ctx, RoleAdmin, PermProductRead) {
		t.Error("expected HasPermissionCtx to return true for admin role")
	}
}

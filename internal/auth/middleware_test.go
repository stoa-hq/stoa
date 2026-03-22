package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestSetAPIKeyContext_StoreKey(t *testing.T) {
	customerID := uuid.New()
	apiKey := &APIKey{
		ID:          uuid.New(),
		KeyType:     "store",
		CustomerID:  &customerID,
		Permissions: []Permission{PermStoreProductRead, PermStoreCartManage},
	}

	ctx := setAPIKeyContext(context.Background(), apiKey)

	if got := UserID(ctx); got != customerID {
		t.Errorf("UserID = %s, want %s (customer_id)", got, customerID)
	}
	if got := UserType(ctx); got != "customer" {
		t.Errorf("UserType = %q, want %q", got, "customer")
	}
	if got := UserRole(ctx); got != RoleCustomer {
		t.Errorf("UserRole = %q, want %q", got, RoleCustomer)
	}
	perms := UserPermissions(ctx)
	if len(perms) != 2 {
		t.Fatalf("UserPermissions length = %d, want 2", len(perms))
	}
	if perms[0] != PermStoreProductRead {
		t.Errorf("UserPermissions[0] = %q, want %q", perms[0], PermStoreProductRead)
	}
}

func TestSetAPIKeyContext_AdminKey_WithCreatedBy(t *testing.T) {
	createdBy := uuid.New()
	apiKeyID := uuid.New()
	apiKey := &APIKey{
		ID:          apiKeyID,
		KeyType:     "admin",
		CreatedBy:   &createdBy,
		Permissions: []Permission{PermProductRead},
	}

	ctx := setAPIKeyContext(context.Background(), apiKey)

	if got := UserID(ctx); got != createdBy {
		t.Errorf("UserID = %s, want %s (created_by)", got, createdBy)
	}
	if got := UserType(ctx); got != "api_key" {
		t.Errorf("UserType = %q, want %q", got, "api_key")
	}
	if got := UserRole(ctx); got != RoleAPIClient {
		t.Errorf("UserRole = %q, want %q", got, RoleAPIClient)
	}
	perms := UserPermissions(ctx)
	if len(perms) != 1 || perms[0] != PermProductRead {
		t.Errorf("UserPermissions = %v, want [%s]", perms, PermProductRead)
	}
}

func TestSetAPIKeyContext_AdminKey_NoCreatedBy(t *testing.T) {
	apiKeyID := uuid.New()
	apiKey := &APIKey{
		ID:          apiKeyID,
		KeyType:     "admin",
		CreatedBy:   nil,
		Permissions: []Permission{PermProductRead, PermOrderRead},
	}

	ctx := setAPIKeyContext(context.Background(), apiKey)

	// Without created_by, should use apiKey.ID as user ID.
	if got := UserID(ctx); got != apiKeyID {
		t.Errorf("UserID = %s, want %s (apiKey.ID)", got, apiKeyID)
	}
	if got := UserType(ctx); got != "api_key" {
		t.Errorf("UserType = %q, want %q", got, "api_key")
	}
	if got := UserRole(ctx); got != RoleAPIClient {
		t.Errorf("UserRole = %q, want %q", got, RoleAPIClient)
	}
}

func TestSetAPIKeyContext_StoreKey_NilCustomerID_FallsBackToAdmin(t *testing.T) {
	// Edge case: key_type="store" but customer_id is nil — treated as admin key.
	apiKeyID := uuid.New()
	apiKey := &APIKey{
		ID:          apiKeyID,
		KeyType:     "store",
		CustomerID:  nil,
		Permissions: []Permission{PermStoreProductRead},
	}

	ctx := setAPIKeyContext(context.Background(), apiKey)

	if got := UserRole(ctx); got != RoleAPIClient {
		t.Errorf("UserRole = %q, want %q (fallback to admin path)", got, RoleAPIClient)
	}
	if got := UserID(ctx); got != apiKeyID {
		t.Errorf("UserID = %s, want %s", got, apiKeyID)
	}
}

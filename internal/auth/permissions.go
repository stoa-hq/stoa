package auth

import "context"

type Role string

const (
	RoleSuperAdmin Role = "super_admin"
	RoleAdmin      Role = "admin"
	RoleManager    Role = "manager"
	RoleCustomer   Role = "customer"
	RoleAPIClient  Role = "api_client"
)

type Permission string

const (
	// Products
	PermProductCreate Permission = "products.create"
	PermProductRead   Permission = "products.read"
	PermProductUpdate Permission = "products.update"
	PermProductDelete Permission = "products.delete"

	// Categories
	PermCategoryCreate Permission = "categories.create"
	PermCategoryRead   Permission = "categories.read"
	PermCategoryUpdate Permission = "categories.update"
	PermCategoryDelete Permission = "categories.delete"

	// Customers
	PermCustomerCreate Permission = "customers.create"
	PermCustomerRead   Permission = "customers.read"
	PermCustomerUpdate Permission = "customers.update"
	PermCustomerDelete Permission = "customers.delete"

	// Orders
	PermOrderCreate Permission = "orders.create"
	PermOrderRead   Permission = "orders.read"
	PermOrderUpdate Permission = "orders.update"
	PermOrderDelete Permission = "orders.delete"

	// Media
	PermMediaCreate Permission = "media.create"
	PermMediaRead   Permission = "media.read"
	PermMediaDelete Permission = "media.delete"

	// Settings
	PermSettingsRead   Permission = "settings.read"
	PermSettingsUpdate Permission = "settings.update"

	// Plugins
	PermPluginManage Permission = "plugins.manage"

	// Audit
	PermAuditRead Permission = "audit.read"

	// Discounts
	PermDiscountCreate Permission = "discounts.create"
	PermDiscountRead   Permission = "discounts.read"
	PermDiscountUpdate Permission = "discounts.update"
	PermDiscountDelete Permission = "discounts.delete"

	// Shipping
	PermShippingCreate Permission = "shipping.create"
	PermShippingRead   Permission = "shipping.read"
	PermShippingUpdate Permission = "shipping.update"
	PermShippingDelete Permission = "shipping.delete"

	// Payment
	PermPaymentCreate Permission = "payment.create"
	PermPaymentRead   Permission = "payment.read"
	PermPaymentUpdate Permission = "payment.update"
	PermPaymentDelete Permission = "payment.delete"

	// Tax
	PermTaxCreate Permission = "tax.create"
	PermTaxRead   Permission = "tax.read"
	PermTaxUpdate Permission = "tax.update"
	PermTaxDelete Permission = "tax.delete"

	// API Keys
	PermAPIKeysManage Permission = "api_keys.manage"

	// Store permissions (for customer API keys)
	PermStoreProductRead   Permission = "store.products.read"
	PermStoreCartManage    Permission = "store.cart.manage"
	PermStoreCheckout      Permission = "store.checkout"
	PermStoreAccountRead   Permission = "store.account.read"
	PermStoreAccountUpdate Permission = "store.account.update"
	PermStoreOrdersRead    Permission = "store.orders.read"
)

var rolePermissions = map[Role][]Permission{
	RoleSuperAdmin: allPermissions(),
	RoleAdmin:      allPermissions(),
	RoleManager: {
		PermProductCreate, PermProductRead, PermProductUpdate,
		PermCategoryRead,
		PermCustomerRead,
		PermOrderRead, PermOrderUpdate,
		PermMediaCreate, PermMediaRead,
		PermDiscountRead,
		PermShippingRead,
		PermPaymentRead,
		PermTaxRead,
		PermAPIKeysManage,
	},
	RoleCustomer: {},
	RoleAPIClient: {},
}

func RolePermissions(role Role) []Permission {
	return rolePermissions[role]
}

func HasPermission(role Role, perm Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

// HasPermissionCtx checks permissions for API clients using context-stored
// permissions, falling back to role-based permissions for other roles.
// Store API keys (role=customer with context permissions) also use
// context-stored permissions.
func HasPermissionCtx(ctx context.Context, role Role, perm Permission) bool {
	if role == RoleAPIClient {
		for _, p := range UserPermissions(ctx) {
			if p == perm {
				return true
			}
		}
		return false
	}
	// Store API keys are authenticated as RoleCustomer but carry
	// explicit permissions in context.
	if role == RoleCustomer {
		if ctxPerms := UserPermissions(ctx); len(ctxPerms) > 0 {
			for _, p := range ctxPerms {
				if p == perm {
					return true
				}
			}
			return false
		}
	}
	return HasPermission(role, perm)
}

// AllStorePermissions returns all store-scoped permissions.
func AllStorePermissions() []Permission {
	return []Permission{
		PermStoreProductRead,
		PermStoreCartManage,
		PermStoreCheckout,
		PermStoreAccountRead,
		PermStoreAccountUpdate,
		PermStoreOrdersRead,
	}
}

// IsStorePermission returns true if the permission is store-scoped.
func IsStorePermission(p Permission) bool {
	for _, sp := range AllStorePermissions() {
		if p == sp {
			return true
		}
	}
	return false
}

func allPermissions() []Permission {
	return []Permission{
		PermProductCreate, PermProductRead, PermProductUpdate, PermProductDelete,
		PermCategoryCreate, PermCategoryRead, PermCategoryUpdate, PermCategoryDelete,
		PermCustomerCreate, PermCustomerRead, PermCustomerUpdate, PermCustomerDelete,
		PermOrderCreate, PermOrderRead, PermOrderUpdate, PermOrderDelete,
		PermMediaCreate, PermMediaRead, PermMediaDelete,
		PermSettingsRead, PermSettingsUpdate,
		PermPluginManage,
		PermAuditRead,
		PermDiscountCreate, PermDiscountRead, PermDiscountUpdate, PermDiscountDelete,
		PermShippingCreate, PermShippingRead, PermShippingUpdate, PermShippingDelete,
		PermPaymentCreate, PermPaymentRead, PermPaymentUpdate, PermPaymentDelete,
		PermTaxCreate, PermTaxRead, PermTaxUpdate, PermTaxDelete,
		PermAPIKeysManage,
	}
}

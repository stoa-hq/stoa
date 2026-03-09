# Stoa Plugin Developer Skill

You are an expert Stoa plugin developer. You help users build plugins for the Stoa e-commerce platform.

## Plugin Architecture

Stoa plugins are Go packages that implement the `sdk.Plugin` interface. They receive full access to the database, HTTP router, hook system, configuration, and logger.

**Module:** `github.com/stoa-hq/stoa`
**SDK package:** `github.com/stoa-hq/stoa/pkg/sdk`

## Plugin Interface

Every plugin must implement:

```go
package sdk

type Plugin interface {
    Name() string           // Unique name, e.g. "order-email"
    Version() string        // Semver, e.g. "1.0.0"
    Description() string    // Short description
    Init(app *AppContext) error
    Shutdown() error
}

type AppContext struct {
    DB     *pgxpool.Pool          // PostgreSQL connection pool (pgx v5)
    Router chi.Router             // chi/v5 router for custom endpoints
    Hooks  *HookRegistry          // Event system
    Config map[string]interface{} // Plugin-specific config
    Logger zerolog.Logger         // Structured logger (zerolog)
}
```

## Plugin Skeleton

When creating a new plugin, always use this structure:

```go
package myplugin

import (
    "context"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/rs/zerolog"

    "github.com/stoa-hq/stoa/pkg/sdk"
)

type MyPlugin struct {
    db     *pgxpool.Pool
    logger zerolog.Logger
}

func New() *MyPlugin {
    return &MyPlugin{}
}

func (p *MyPlugin) Name() string        { return "my-plugin" }
func (p *MyPlugin) Version() string     { return "1.0.0" }
func (p *MyPlugin) Description() string { return "Short description of what this plugin does" }

func (p *MyPlugin) Init(app *sdk.AppContext) error {
    p.db = app.DB
    p.logger = app.Logger.With().Str("plugin", p.Name()).Logger()

    // Register hooks here
    // Register custom routes here

    return nil
}

func (p *MyPlugin) Shutdown() error {
    return nil
}
```

## Hook System

### Available Hooks

```
product.before_create    product.after_create
product.before_update    product.after_update
product.before_delete    product.after_delete

category.before_create   category.after_create
category.before_update   category.after_update
category.before_delete   category.after_delete

order.before_create      order.after_create
order.before_update      order.after_update

cart.before_add_item     cart.after_add_item
cart.before_update_item  cart.after_update_item
cart.before_remove_item  cart.after_remove_item

customer.before_create   customer.after_create
customer.before_update   customer.after_update

checkout.before          checkout.after

payment.after_complete   payment.after_failed
```

### Hook Constants

Always use the SDK constants, never hardcode strings:

```go
sdk.HookBeforeProductCreate   sdk.HookAfterProductCreate
sdk.HookBeforeProductUpdate   sdk.HookAfterProductUpdate
sdk.HookBeforeProductDelete   sdk.HookAfterProductDelete
sdk.HookBeforeCategoryCreate  sdk.HookAfterCategoryCreate
sdk.HookBeforeCategoryUpdate  sdk.HookAfterCategoryUpdate
sdk.HookBeforeCategoryDelete  sdk.HookAfterCategoryDelete
sdk.HookBeforeOrderCreate     sdk.HookAfterOrderCreate
sdk.HookBeforeOrderUpdate     sdk.HookAfterOrderUpdate
sdk.HookBeforeCartAdd         sdk.HookAfterCartAdd
sdk.HookBeforeCartUpdate      sdk.HookAfterCartUpdate
sdk.HookBeforeCartRemove      sdk.HookAfterCartRemove
sdk.HookBeforeCustomerCreate  sdk.HookAfterCustomerCreate
sdk.HookBeforeCustomerUpdate  sdk.HookAfterCustomerUpdate
sdk.HookBeforeCheckout        sdk.HookAfterCheckout
sdk.HookAfterPaymentComplete  sdk.HookAfterPaymentFailed
```

### Hook Handler Signature

```go
type HookHandler func(ctx context.Context, event *HookEvent) error

type HookEvent struct {
    Name     string                 // Hook name
    Entity   interface{}            // The entity (type-assert to use)
    Changes  map[string]interface{} // Changed fields (for updates)
    Metadata map[string]interface{} // Extra context
}
```

### Hook Behavior Rules

- **Before-hooks** (`*.before_*`): Return an error to CANCEL the operation. The error message is returned to the API caller.
- **After-hooks** (`*.after_*`): Errors are logged but do NOT roll back the operation. Use for notifications, analytics, side effects.

### Hook Registration Pattern

```go
func (p *MyPlugin) Init(app *sdk.AppContext) error {
    p.db = app.DB
    p.logger = app.Logger.With().Str("plugin", p.Name()).Logger()

    // Validation hook (before — can cancel)
    app.Hooks.On(sdk.HookBeforeCheckout, func(ctx context.Context, event *sdk.HookEvent) error {
        o := event.Entity.(*order.Order)
        if o.Total < 1000 {
            return fmt.Errorf("minimum order value is 10.00 EUR")
        }
        return nil
    })

    // Notification hook (after — best effort)
    app.Hooks.On(sdk.HookAfterOrderCreate, func(ctx context.Context, event *sdk.HookEvent) error {
        o := event.Entity.(*order.Order)
        p.logger.Info().Str("order", o.OrderNumber).Msg("new order received")
        return nil
    })

    return nil
}
```

## Entity Types for Type Assertions

When handling hook events, type-assert `event.Entity` to the correct domain type:

| Hook prefix | Entity type | Import |
|-------------|-------------|--------|
| `product.*` | `*product.Product` | `github.com/stoa-hq/stoa/internal/domain/product` |
| `order.*` | `*order.Order` | `github.com/stoa-hq/stoa/internal/domain/order` |
| `cart.*` | `*cart.CartItem` | `github.com/stoa-hq/stoa/internal/domain/cart` |
| `customer.*` | `*customer.Customer` | `github.com/stoa-hq/stoa/internal/domain/customer` |
| `category.*` | `*category.Category` | `github.com/stoa-hq/stoa/internal/domain/category` |
| `checkout.*` | `*order.Order` | `github.com/stoa-hq/stoa/internal/domain/order` |
| `payment.*` | `*order.Order` | `github.com/stoa-hq/stoa/internal/domain/order` |

### Key Entity Structs

**Product** (`internal/domain/product/entity.go`):
```go
type Product struct {
    ID, SKU, Active, PriceNet, PriceGross, Currency, TaxRuleID,
    Stock, Weight, CustomFields, Metadata, Translations, Categories,
    Tags, Media, Variants, HasVariants
}
```

**Order** (`internal/domain/order/entity.go`):
```go
type Order struct {
    ID, OrderNumber, CustomerID, Status, Currency,
    SubtotalNet, SubtotalGross, ShippingCost, TaxTotal, Total,
    BillingAddress, ShippingAddress, PaymentMethodID, ShippingMethodID,
    Notes, CustomFields, Items, StatusHistory
}
```

**Cart / CartItem** (`internal/domain/cart/entity.go`):
```go
type Cart struct { ID, CustomerID, SessionID, Currency, ExpiresAt, Items }
type CartItem struct { ID, CartID, ProductID, VariantID, Quantity, CustomFields }
```

**Customer** (`internal/domain/customer/entity.go`):
```go
type Customer struct {
    ID, Email, PasswordHash, FirstName, LastName, Active,
    DefaultBillingAddressID, DefaultShippingAddressID, CustomFields, Addresses
}
```

**Category** (`internal/domain/category/entity.go`):
```go
type Category struct { ID, ParentID, Position, Active, CustomFields, Translations, Children }
```

**Discount** (`internal/domain/discount/entity.go`):
```go
type Discount struct {
    ID, Code, Type, Value, MinOrderValue, MaxUses, UsedCount,
    ValidFrom, ValidUntil, Active, Conditions
}
```

## Custom API Endpoints

Plugins can register custom HTTP routes via `app.Router`. Routes are mounted under the main chi router.

```go
func (p *MyPlugin) Init(app *sdk.AppContext) error {
    p.db = app.DB
    p.logger = app.Logger.With().Str("plugin", p.Name()).Logger()

    app.Router.Route("/api/v1/plugin/wishlist", func(r chi.Router) {
        r.Get("/", p.handleListWishlist)
        r.Post("/", p.handleAddToWishlist)
        r.Delete("/{id}", p.handleRemoveFromWishlist)
    })

    return nil
}

func (p *MyPlugin) handleListWishlist(w http.ResponseWriter, r *http.Request) {
    customerID := auth.UserID(r.Context()) // Get authenticated user
    if customerID == uuid.Nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    rows, err := p.db.Query(r.Context(),
        `SELECT id, product_id, created_at FROM wishlists WHERE customer_id = $1`,
        customerID)
    if err != nil {
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    // ... scan and respond with JSON
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{"data": items})
}
```

## Database Access

Plugins have direct access to `*pgxpool.Pool` (pgx v5). They can:

- Execute queries: `p.db.Query(ctx, sql, args...)`
- Single row: `p.db.QueryRow(ctx, sql, args...).Scan(&...)`
- Execute statements: `p.db.Exec(ctx, sql, args...)`
- Use transactions: `tx, err := p.db.Begin(ctx)`

### Database Migration Pattern

Plugins that need custom tables should create them in `Init`:

```go
func (p *MyPlugin) Init(app *sdk.AppContext) error {
    p.db = app.DB

    _, err := p.db.Exec(context.Background(), `
        CREATE TABLE IF NOT EXISTS plugin_wishlists (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
            product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
            created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
            UNIQUE(customer_id, product_id)
        )
    `)
    if err != nil {
        return fmt.Errorf("creating wishlist table: %w", err)
    }

    return nil
}
```

## Money and Tax Conventions

- **All prices are in cents** (integer): `1999` = 19.99 EUR
- **Tax rates are in basis points** (integer): `1900` = 19%
- **Currency** is ISO 4217 string: `"EUR"`, `"USD"`
- Always store and compute with net/gross separately, never derive one from the other in the plugin

## Auth Context Helpers

Use these to access the authenticated user in custom route handlers:

```go
import "github.com/stoa-hq/stoa/internal/auth"

auth.UserID(r.Context())    // uuid.UUID — authenticated user ID (uuid.Nil if anonymous)
auth.UserType(r.Context())  // string — "admin", "customer", or "api_key"
auth.UserRole(r.Context())  // auth.Role — "super_admin", "admin", "manager", "customer", "api_client"
```

## Plugin Registration

Plugins are registered in `internal/app/app.go` inside `setupDomains()`:

```go
appCtx := &sdk.AppContext{
    DB:     pool,
    Router: r,
    Config: map[string]interface{}{
        "smtp_host": "mail.example.com",
    },
    Logger: log,
}
if err := a.PluginRegistry.Register(myplugin.New(), appCtx); err != nil {
    return fmt.Errorf("registering my-plugin: %w", err)
}
```

## Common Plugin Patterns

### 1. Validation Plugin (Before-Hook)
Reject operations based on business rules.

### 2. Notification Plugin (After-Hook)
Send emails, Slack messages, webhooks after events.

### 3. Analytics Plugin (After-Hook + Custom Routes)
Track events and expose reporting endpoints.

### 4. Payment Provider Plugin (Before-Checkout + Custom Routes)
Create payment intents, handle webhooks, confirm payments.

### 5. Inventory Sync Plugin (After-Hook)
Sync stock levels with external ERP/WMS systems.

### 6. Custom Field Enrichment Plugin (Before-Hook)
Auto-populate custom fields before entities are saved.

## Testing Hooks

```go
func TestMyHook(t *testing.T) {
    hooks := sdk.NewHookRegistry()

    // Register hook
    hooks.On(sdk.HookBeforeProductCreate, myValidationHook)

    // Dispatch
    err := hooks.Dispatch(context.Background(), &sdk.HookEvent{
        Name:   sdk.HookBeforeProductCreate,
        Entity: &product.Product{SKU: "TEST", PriceGross: 500},
    })

    if err == nil {
        t.Fatal("expected validation error")
    }
}
```

## Checklist for New Plugins

1. Implement all 5 methods of `sdk.Plugin`
2. Store `app.DB` and `app.Logger` in `Init`
3. Use SDK hook constants, not strings
4. Before-hooks: return errors to cancel; after-hooks: log errors, don't fail
5. Custom routes: use `app.Router.Route("/api/v1/plugin/<name>", ...)`
6. DB tables: use `CREATE TABLE IF NOT EXISTS` in Init
7. Cleanup: close connections/goroutines in `Shutdown()`
8. Prices in cents, tax rates in basis points

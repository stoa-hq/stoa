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
    DB          *pgxpool.Pool          // PostgreSQL connection pool (pgx v5)
    Router      chi.Router             // chi/v5 router for custom endpoints
    AssetRouter chi.Router             // Mounted at /plugins/{name}/assets/
    Hooks       *HookRegistry          // Event system
    Config      map[string]interface{} // Plugin-specific config
    Logger      zerolog.Logger         // Structured logger (zerolog)
    Auth        *AuthHelper            // Auth middleware + context helpers
}

type AuthHelper struct {
    OptionalAuth func(http.Handler) http.Handler  // Extracts auth if present, never blocks
    Required     func(http.Handler) http.Handler  // Requires valid token, returns 401
    UserID       func(ctx context.Context) uuid.UUID // Authenticated user ID
    UserType     func(ctx context.Context) string    // "admin", "customer", "api_key"
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
    Notes, GuestToken, CustomFields, Items, StatusHistory
}
```

`GuestToken` is a UUID string set for guest (unauthenticated) orders. It is `""` for authenticated customer orders.

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

**IMPORTANT**: The plugin router is the ROOT chi router. It does NOT inherit Stoa's store middleware (`OptionalAuth`). You MUST apply auth middleware explicitly using `app.Auth.Required` or `app.Auth.OptionalAuth`.

```go
func (p *MyPlugin) Init(app *sdk.AppContext) error {
    p.db = app.DB
    p.auth = app.Auth
    p.logger = app.Logger.With().Str("plugin", p.Name()).Logger()

    // Store-facing routes: ALWAYS apply auth middleware
    app.Router.Route("/api/v1/store/wishlist", func(r chi.Router) {
        r.Use(app.Auth.Required) // Requires authentication
        r.Get("/", p.handleListWishlist)
        r.Post("/", p.handleAddToWishlist)
        r.Delete("/{id}", p.handleRemoveFromWishlist)
    })

    // Webhook routes: no auth middleware (verified by signature/secret)
    app.Router.Post("/plugins/myplugin/webhook", p.handleWebhook)

    return nil
}

func (p *MyPlugin) handleListWishlist(w http.ResponseWriter, r *http.Request) {
    customerID := p.auth.UserID(r.Context()) // Use AuthHelper, not internal/auth
    if customerID == uuid.Nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    // Always filter by customer_id to prevent IDOR attacks
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

Use the `AuthHelper` from `AppContext` to access the authenticated user. **Do not import `internal/auth` directly** — use `app.Auth` instead:

```go
// In Init: store the AuthHelper
p.auth = app.Auth

// In handlers:
userID := p.auth.UserID(r.Context())     // uuid.UUID — uuid.Nil if anonymous
userType := p.auth.UserType(r.Context()) // "admin", "customer", "api_key", or ""

// Apply middleware to routes:
r.Use(app.Auth.Required)     // Requires valid token, returns 401 otherwise
r.Use(app.Auth.OptionalAuth) // Extracts auth if present, never blocks
```

### Security Rules for Store-Facing Endpoints

1. **Always apply auth middleware** — the plugin router does NOT inherit `/api/v1/store/*` middleware
2. **Always verify ownership** — filter DB queries by `customer_id` for authenticated users, or by `guest_token` for guest orders (see Guest Checkout below)
3. **Never leak internal errors** — return generic error messages to API consumers

### Guest Checkout Ownership

For endpoints that support guest users (e.g. payment), use `OptionalAuth` and verify ownership with either `customer_id` or `guest_token`:

```go
r.Use(app.Auth.OptionalAuth) // NOT Required — guests have no token

func (p *Plugin) handleAction(w http.ResponseWriter, r *http.Request) {
    userID := p.auth.UserID(r.Context())
    var query string
    var args []interface{}

    if userID != uuid.Nil {
        // Authenticated customer
        query = `SELECT ... FROM orders WHERE id = $1 AND customer_id = $2`
        args = []interface{}{orderID, userID}
    } else {
        // Guest — require guest_token
        if req.GuestToken == "" {
            writeError(w, http.StatusUnauthorized, "authentication or guest token required")
            return
        }
        query = `SELECT ... FROM orders WHERE id = $1 AND guest_token = $2 AND customer_id IS NULL`
        args = []interface{}{orderID, req.GuestToken}
    }
    // ...
}
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

## MCP Store Tools (Optional)

Plugins can register additional tools on the Store MCP server by implementing `sdk.MCPStorePlugin`:

```go
type MCPStorePlugin interface {
    Plugin
    RegisterStoreMCPTools(server any, client StoreAPIClient)
}

type StoreAPIClient interface {
    Get(path string) ([]byte, error)   // Only /api/v1/store/* paths allowed
    Post(path string, body interface{}) ([]byte, error)
}
```

### MCP Tool Pattern

```go
// toolAdder is satisfied by both *server.MCPServer and *mcp.ScopedMCPServer.
type toolAdder interface {
    AddTool(mcp.Tool, server.ToolHandlerFunc)
}

func (p *Plugin) RegisterStoreMCPTools(srv any, client sdk.StoreAPIClient) {
    s := srv.(toolAdder)  // Interface assertion — NOT srv.(*server.MCPServer)

    tool := mcp.NewTool("store_myplugin_action",  // MUST use prefix store_{pluginName}_
        mcp.WithDescription("Description for AI agents"),
        mcp.WithString("order_id", mcp.Required()),
    )
    s.AddTool(tool, func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        data, err := client.Post("/api/v1/store/myplugin/action", map[string]interface{}{
            "order_id": req.GetString("order_id", ""),
        })
        if err != nil {
            return mcp.NewToolResultError("action failed"), nil  // Sanitize errors!
        }
        return mcp.NewToolResultText(string(data)), nil
    })
}
```

### MCP Security Rules

- **Tool names MUST use prefix `store_{pluginName}_`** — enforced by ScopedMCPServer, panics on violation
- **Use interface assertion** `srv.(toolAdder)` — NOT `srv.(*server.MCPServer)`. The server passes a scoped wrapper
- **StoreAPIClient is restricted** to `/api/v1/store/*` paths — admin endpoints are blocked
- **Sanitize error messages** — return generic errors, never `err.Error()` directly to MCP consumers
- **Panic recovery** — if registration panics, plugin is skipped and MCP server continues

## Webhook Handler Security

When writing webhook handlers (Stripe, PayPal, etc.):

1. **Verify signatures** — always validate webhook signatures with the raw body before processing
2. **Use `context.Background()` with timeout** for goroutines — not `r.Context()` which is canceled when the handler returns
3. **Implement idempotency** — use `ON CONFLICT (provider_reference) DO NOTHING` to handle duplicate webhook deliveries
4. **Return 204 quickly** — process webhooks async in goroutines to avoid Stripe/provider timeouts

```go
// WRONG — context canceled after handler returns
go handleEvent(r.Context(), event, db)

// CORRECT — detached context with timeout
bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
go func() {
    defer cancel()
    handleEvent(bgCtx, event, db)
}()
w.WriteHeader(http.StatusNoContent)
```

## UI Extensions (Optional)

Plugins can extend Admin Panel and Storefront UI by implementing `sdk.UIPlugin`:

```go
type UIPlugin interface {
    Plugin
    UIExtensions() []UIExtension
}
```

### Schema-based Form (simple settings)

```go
func (p *Plugin) UIExtensions() []sdk.UIExtension {
    return []sdk.UIExtension{
        {
            ID:   "myplugin_settings",
            Slot: "admin:payment:settings",
            Type: "schema",
            Schema: &sdk.UISchema{
                Fields: []sdk.UISchemaField{
                    {Key: "api_key", Type: "password", Label: map[string]string{"en": "API Key", "de": "API-Schlüssel"}},
                    {Key: "mode", Type: "select", Label: map[string]string{"en": "Mode"},
                        Options: []sdk.UISelectOption{
                            {Value: "test", Label: map[string]string{"en": "Test"}},
                            {Value: "live", Label: map[string]string{"en": "Live"}},
                        },
                    },
                },
                SubmitURL: "/api/v1/admin/plugins/myplugin/settings",
                LoadURL:   "/api/v1/admin/plugins/myplugin/settings",
            },
        },
    }
}
```

Field types: `text`, `password`, `toggle`, `select`, `number`, `textarea`.
Labels support i18n via `map[string]string` (locale → text).

### Web Component (complex UI)

```go
{
    ID:   "myplugin_checkout",
    Slot: "storefront:checkout:payment",
    Type: "component",
    Component: &sdk.UIComponent{
        TagName:         "stoa-myplugin-checkout",  // MUST start with stoa-{pluginName}-
        ScriptURL:       "/plugins/myplugin/assets/checkout.js",
        Integrity:       "sha256-...",
        ExternalScripts: []string{"https://js.example.com/v3/", "https://api.example.com"},
    },
}
```

Web components receive `context` (slot-specific data) and `apiClient` (scoped HTTP client) as properties.
Dispatch `plugin-event` CustomEvents to communicate with the host page.

**Rendering: Light DOM with scoped CSS.** Web components render in **Light DOM** (no Shadow DOM). Third-party services like Stripe require direct DOM access for iframes and cannot work inside Shadow DOM. Use scoped CSS class prefixes (e.g. `.stoa-myplugin-`) to avoid style collisions.

**ExternalScripts** are added to the Content-Security-Policy `script-src`, `frame-src`, AND `connect-src` directives. Include all external domains your component needs (e.g. both `js.stripe.com` and `api.stripe.com` for Stripe).

**apiClient** already unwraps the API response envelope — `await this.apiClient.post(...)` returns `{ client_secret, ... }` directly, NOT `{ data: { client_secret } }`.

### Serving Embedded Assets

The core mounts `app.AssetRouter` at `/plugins/{name}/assets/` with path stripping already applied. Plugins just serve the filesystem directly:

```go
//go:embed frontend/dist
var assetsFS embed.FS

func (p *Plugin) Init(app *sdk.AppContext) error {
    sub, _ := fs.Sub(assetsFS, "frontend/dist")
    app.AssetRouter.Handle("/*", http.FileServerFS(sub))
    return nil
}
```

### Available Slots

| Slot | Location | SPA |
|------|----------|-----|
| `storefront:checkout:payment` | After payment selection | Storefront |
| `storefront:checkout:after_order` | After order confirmation | Storefront |
| `admin:payment:settings` | Payment method detail page | Admin |
| `admin:sidebar` | Sidebar navigation | Admin |
| `admin:dashboard:widget` | Dashboard widgets | Admin |

### UI Extension Validation Rules

- Slot must start with `storefront:` or `admin:`
- Tag names must use `stoa-{pluginName}-` prefix
- URLs must not contain `..` (path traversal) or absolute URLs
- Invalid extensions are skipped with a warning at startup

## Checklist for New Plugins

1. Implement all 5 methods of `sdk.Plugin`
2. Store `app.DB`, `app.Logger`, and `app.Auth` in `Init`
3. Use SDK hook constants, not strings
4. Before-hooks: return errors to cancel; after-hooks: log errors, don't fail
5. **Apply `app.Auth.Required` or `app.Auth.OptionalAuth`** to store-facing routes
6. **Verify ownership** — filter by `customer_id` (authenticated) or `guest_token` (guest) in store-facing DB queries
7. Custom routes: use `app.Router.Route("/api/v1/store/<name>", ...)`
8. DB tables: use `CREATE TABLE IF NOT EXISTS` in Init
9. Cleanup: close connections/goroutines in `Shutdown()`
10. Prices in cents, tax rates in basis points
11. MCP tool names: prefix `store_{pluginName}_*`
12. MCP errors: sanitize — never expose internal details
13. Webhooks: verify signatures, use background context, implement idempotency
14. UI extensions: implement `UIPlugin` if the plugin needs frontend UI
15. Web Component tag names: `stoa-{pluginName}-*` prefix required
16. Asset serving: use `app.AssetRouter` for embedded frontend files

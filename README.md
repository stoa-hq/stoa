# Stoa

A lightweight, open-source headless e-commerce platform built with Go. The system ships as a single binary with both the admin panel and the storefront embedded.

## Features

- **Headless Architecture** -- REST API (JSON)
- **Single Binary** -- Go backend with embedded SvelteKit frontends (Admin + Storefront)
- **MCP Servers** -- AI agents can shop in and manage the store via the Model Context Protocol
- **Plugin System** -- Extensible via hooks and custom API endpoints
- **Multi-language** -- Translation tables with locale-based API
- **Property Groups & Variants** -- Color, size, etc. with automatic combination generation
- **Full-text Search** -- PostgreSQL-based
- **RBAC** -- Role-based access control with granular API key permissions

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Docker + Docker Compose | latest | Database (and optional app container) |
| Go | 1.23+ | Build backend (local development only) |
| Node.js | 20+ | Build frontends (local development only) |
| PostgreSQL | 16+ | Database (provided via Docker) |

---

## Quick Start with Docker (recommended)

This is the easiest way to run the entire platform locally. All you need is Docker.

### 1. Clone the repository

```bash
git clone <repository-url>
cd stoa
```

### 2. Create configuration

```bash
cp config.example.yaml config.yaml
```

The default values work out of the box with Docker Compose -- no changes required.

### 3. Start everything

```bash
docker compose up -d
```

This starts PostgreSQL and the Stoa application. On the first run the Docker image is built (including admin and storefront frontends), which takes a few minutes.

### 4. Set up the database

```bash
# Run migrations (create tables)
docker compose exec stoa ./stoa migrate up

# Create an admin user
docker compose exec stoa ./stoa admin create --email admin@example.com --password your-password

# Optional: load demo data (products, categories, etc.)
docker compose exec stoa ./stoa seed --demo
```

### 5. Done!

| What | URL |
|------|-----|
| Storefront | http://localhost:8080 |
| Admin Panel | http://localhost:8080/admin |
| API Health Check | http://localhost:8080/api/v1/health |

Log into the admin panel with the credentials from step 4.

### Stopping and Restarting

```bash
# Stop (data is preserved)
docker compose down

# Stop and delete all data
docker compose down -v

# Restart
docker compose up -d
```

---

## Local Development (without Docker for the app)

For working on the codebase it is more convenient to run only PostgreSQL via Docker and execute the app directly.

### 1. Start PostgreSQL

```bash
docker compose up -d postgres
```

### 2. Create configuration

```bash
cp config.example.yaml config.yaml
```

### 3. Set up the database

```bash
go run ./cmd/stoa migrate up
go run ./cmd/stoa admin create --email admin@example.com --password your-password
go run ./cmd/stoa seed --demo   # optional
```

### 4. Build frontends

Both admin and storefront are SvelteKit applications embedded into the Go binary via `//go:embed`. They must be built before the first run:

```bash
# Admin panel
cd admin && npm install && npm run build && cd ..

# Storefront
cd storefront && npm install && npm run build && cd ..
```

> **Important:** After every change to the frontends you must run `npm run build` AND rebuild the Go binary, because the frontends are statically embedded into the binary.

### 5. Start the backend

```bash
go run ./cmd/stoa serve
```

Or as a compiled binary:

```bash
go build -o stoa ./cmd/stoa
./stoa serve
```

### Frontend Development with Hot-Reload

For frontend development you can start the Vite dev servers, which provide hot-reload:

```bash
# Admin panel (port 5174)
cd admin && npm run dev

# Storefront (port 5173)
cd storefront && npm run dev
```

The dev servers communicate with the Go backend on port 8080 via the API. Make sure the backend is running.

---

## Makefile Commands

```bash
make build              # Build frontends + compile Go binary
make run                # build + start
make test               # Run Go tests
make test-race          # Tests with race detector
make lint               # Run linters (golangci-lint + go vet)
make docker-up          # docker compose up -d
make docker-down        # docker compose down
make admin-dev          # Admin frontend dev server
make storefront-dev     # Storefront dev server
make seed               # Load demo data
make mcp-store-build    # Build Store MCP Server binary
make mcp-admin-build    # Build Admin MCP Server binary
make mcp-store-run      # Build + run Store MCP Server (SSE on :8090)
make mcp-admin-run      # Build + run Admin MCP Server (SSE on :8090)
```

---

## Configuration

All settings are in `config.yaml`. Alternatively they can be overridden via environment variables with the `STOA_` prefix:

```bash
STOA_DATABASE_URL="postgres://user:pass@host:5432/db?sslmode=disable"
STOA_AUTH_JWT_SECRET="a-secure-secret"
STOA_SERVER_PORT=8080
```

### Key Settings

| Setting | Default | Description |
|---------|---------|-------------|
| `server.port` | `8080` | HTTP port |
| `database.url` | `postgres://stoa:secret@localhost:5432/stoa` | PostgreSQL connection string |
| `auth.jwt_secret` | `change-me-in-production` | JWT signing key |
| `media.storage` | `local` | Media storage (`local` or `s3`) |
| `media.local_path` | `./uploads` | Local upload path |
| `i18n.default_locale` | `de-DE` | Default language |
| `payment.encryption_key` | *(required)* | AES-256 key for payment config encryption (32 bytes or 64 hex chars, env: `STOA_PAYMENT_ENCRYPTION_KEY`) |

---

## API Overview

| Area | Path | Authentication |
|------|------|----------------|
| Admin API | `/api/v1/admin/*` | JWT (admin role) or API key with permissions |
| Store API | `/api/v1/store/*` | Public / customer JWT / API key |
| Auth | `/api/v1/auth/*` | None |
| Health | `/api/v1/health` | None |

### Authentication

```bash
# Admin login (JWT)
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email": "admin@example.com", "password": "your-password"}'

# The response contains access_token and refresh_token
# Use access_token in the Authorization header:
curl http://localhost:8080/api/v1/admin/products \
  -H 'Authorization: Bearer <access_token>'

# API key authentication (for MCP servers and integrations):
curl http://localhost:8080/api/v1/admin/products \
  -H 'Authorization: ApiKey ck_your_api_key_here'
```

---

## CLI Commands

```bash
stoa serve                  # Start HTTP server
stoa migrate up             # Run migrations
stoa migrate down           # Roll back last migration
stoa admin create           # Create admin user
  --email admin@example.com
  --password your-password
stoa seed --demo            # Load demo data
stoa plugin list            # List installed plugins
stoa version                # Print version
```

---

## MCP Servers (AI Agent Integration)

Stoa ships with two MCP (Model Context Protocol) servers that allow AI agents -- such as Claude -- to interact with the shop programmatically.

| Server | Binary | Tools | Purpose |
|--------|--------|-------|---------|
| **Store MCP** | `stoa-store-mcp` | 16 | Shopping: browse products, manage cart, checkout |
| **Admin MCP** | `stoa-admin-mcp` | 33 | Management: products, orders, discounts, customers, ... |

### Prerequisites

1. A running Stoa instance
2. An API key with the required permissions

### Create an API Key

API keys are managed through the admin API. Only `super_admin` and `admin` roles can create keys.

```bash
# Login as admin
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"your-password"}' | jq -r '.data.access_token')

# Create an API key with full admin permissions
curl -X POST http://localhost:8080/api/v1/admin/api-keys \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "MCP Admin Key",
    "permissions": [
      "products.read", "products.create", "products.update", "products.delete",
      "orders.read", "orders.update",
      "discounts.read", "discounts.create", "discounts.update", "discounts.delete",
      "customers.read", "customers.update", "customers.delete",
      "categories.read", "categories.create", "categories.update",
      "media.read", "media.delete",
      "shipping.read", "payment.read", "tax.read",
      "audit.read"
    ]
  }'

# Save the "key" field from the response -- it is shown only once!
```

For the Store MCP server, no API key is needed for public endpoints (browsing, cart). An API key or customer JWT is only required for account-related operations.

### Build

```bash
make mcp-store-build    # → bin/stoa-store-mcp
make mcp-admin-build    # → bin/stoa-admin-mcp
```

### Configuration

Both servers are configured via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `STOA_MCP_API_URL` | `http://localhost:8080` | Stoa backend URL |
| `STOA_MCP_API_KEY` | *(empty)* | API key for authentication |
| `STOA_MCP_PORT` | `8090` | HTTP port for SSE server |
| `STOA_MCP_BASE_URL` | `http://localhost:<port>` | Public base URL (for proxied setups) |

### Run

```bash
# Store MCP on port 8091, Admin MCP on port 8092
STOA_MCP_PORT=8091 make mcp-store-run                         # bin/stoa-store-mcp
STOA_MCP_PORT=8092 STOA_MCP_API_KEY=ck_... make mcp-admin-run # bin/stoa-admin-mcp
```

Both servers expose SSE endpoints:
- **SSE stream:** `http://localhost:<port>/sse`
- **Message endpoint:** `http://localhost:<port>/message`

### Use with Claude Code

Add the MCP servers to your Claude Code configuration (`.claude/settings.json` or project settings):

```json
{
  "mcpServers": {
    "stoa-store": {
      "url": "http://localhost:8091/sse"
    },
    "stoa-admin": {
      "url": "http://localhost:8092/sse"
    }
  }
}
```

Once configured, you can interact with the shop in natural language:

- *"Show me all shoes under 50 EUR"*
- *"Add the leather boots to the cart"*
- *"Create a 20% discount code SUMMER for all orders over 50 EUR"*
- *"What are the last 10 orders?"*

### Store MCP Tools (16)

| Category | Tools |
|----------|-------|
| **Products** | `store_list_products`, `store_get_product`, `store_search`, `store_get_categories` |
| **Cart** | `store_create_cart`, `store_get_cart`, `store_add_to_cart`, `store_update_cart_item`, `store_remove_from_cart` |
| **Checkout** | `store_get_shipping_methods`, `store_get_payment_methods`, `store_checkout` |
| **Account** | `store_register`, `store_login`, `store_get_account`, `store_list_orders` |

### Admin MCP Tools (33)

| Category | Tools |
|----------|-------|
| **Products** (8) | `admin_list_products`, `admin_get_product`, `admin_create_product`, `admin_update_product`, `admin_delete_product`, `admin_create_variant`, `admin_update_variant`, `admin_delete_variant` |
| **Orders** (3) | `admin_list_orders`, `admin_get_order`, `admin_update_order_status` |
| **Discounts** (5) | `admin_list_discounts`, `admin_get_discount`, `admin_create_discount`, `admin_update_discount`, `admin_delete_discount` |
| **Customers** (4) | `admin_list_customers`, `admin_get_customer`, `admin_update_customer`, `admin_delete_customer` |
| **Categories** (4) | `admin_list_categories`, `admin_get_category`, `admin_create_category`, `admin_update_category` |
| **Tags** (3) | `admin_list_tags`, `admin_create_tag`, `admin_delete_tag` |
| **Media** (2) | `admin_list_media`, `admin_delete_media` |
| **Config** (3) | `admin_list_shipping_methods`, `admin_list_tax_rules`, `admin_list_payment_methods` |
| **Audit** (1) | `admin_list_audit_log` |

---

## Project Structure

```
stoa/
├── cmd/
│   ├── stoa/               # CLI entry point (main.go)
│   ├── stoa-store-mcp/     # Store MCP Server (shopping)
│   └── stoa-admin-mcp/     # Admin MCP Server (management)
├── internal/
│   ├── app/                # Application bootstrapping
│   ├── config/             # Configuration loading
│   ├── crypto/             # AES-256-GCM encryption helpers
│   ├── server/             # HTTP server, router, middleware
│   ├── auth/               # JWT, RBAC, API keys, permissions
│   ├── database/           # DB connection, migration runner
│   ├── domain/             # Business logic (DDD-style)
│   │   ├── product/        # Products, variants, property groups
│   │   ├── category/       # Categories (tree structure)
│   │   ├── order/          # Orders
│   │   ├── cart/           # Shopping cart
│   │   ├── customer/       # Customer management
│   │   ├── media/          # Media uploads
│   │   ├── discount/       # Discounts
│   │   ├── shipping/       # Shipping methods
│   │   ├── payment/        # Payment methods
│   │   ├── tax/            # Tax rules
│   │   ├── tag/            # Tags
│   │   └── audit/          # Audit log
│   ├── mcp/                # Shared MCP infrastructure
│   │   ├── store/          # Store MCP tools (16)
│   │   └── admin/          # Admin MCP tools (33)
│   ├── admin/              # Embedded admin frontend (//go:embed)
│   ├── storefront/         # Embedded storefront (//go:embed)
│   ├── plugin/             # Plugin registry
│   └── search/             # Search index
├── admin/                  # Admin frontend (SvelteKit)
├── storefront/             # Storefront (SvelteKit)
├── migrations/             # SQL migrations
├── pkg/sdk/                # Plugin SDK
├── Dockerfile
├── docker-compose.yaml
├── Makefile
└── config.example.yaml
```

Every domain follows the same pattern:
- `entity.go` -- Data structures
- `repository.go` -- Interface
- `postgres.go` -- Implementation
- `service.go` -- Business logic
- `handler.go` -- HTTP handlers
- `dto.go` -- Request/response types

---

## Developing Plugins

Stoa includes a Claude Code skill for plugin development. Run `/plugin` in Claude Code to activate it -- it provides the full SDK reference, all hook constants, entity types, and ready-to-use templates for rapid plugin development.

Stoa has a built-in plugin system that lets you extend the platform without modifying core code. Plugins can:

- **React to events** (e.g. send an email after an order)
- **Prevent operations** (e.g. validate before a cart change)
- **Provide custom API endpoints**
- **Access the database directly**

### Plugin Interface

Every plugin implements the `sdk.Plugin` interface from `pkg/sdk`:

```go
package sdk

type Plugin interface {
    Name() string        // Unique name, e.g. "order-email"
    Version() string     // Semver, e.g. "1.0.0"
    Description() string // Short description
    Init(app *AppContext) error   // Called on startup
    Shutdown() error              // Called on shutdown
}
```

In the `Init` method the plugin receives an `AppContext` with everything it needs:

```go
type AppContext struct {
    DB     *pgxpool.Pool       // PostgreSQL connection
    Router chi.Router           // HTTP router for custom endpoints
    Hooks  *HookRegistry        // Event system
    Config map[string]interface{} // Plugin-specific configuration
    Logger zerolog.Logger        // Structured logging
}
```

### Example: Email on New Order

Create a new file, e.g. `plugins/orderemail/plugin.go`:

```go
package orderemail

import (
    "context"
    "fmt"

    "github.com/epoxx-arch/stoa/internal/domain/order"
    "github.com/epoxx-arch/stoa/pkg/sdk"
)

type Plugin struct {
    logger zerolog.Logger
}

func New() *Plugin {
    return &Plugin{}
}

func (p *Plugin) Name() string        { return "order-email" }
func (p *Plugin) Version() string     { return "1.0.0" }
func (p *Plugin) Description() string { return "Sends confirmation emails after orders" }

func (p *Plugin) Init(app *sdk.AppContext) error {
    p.logger = app.Logger

    // Send an email after every new order
    app.Hooks.On(sdk.HookAfterOrderCreate, func(ctx context.Context, event *sdk.HookEvent) error {
        o := event.Entity.(*order.Order)
        p.logger.Info().
            Str("order", o.OrderNumber).
            Msg("sending confirmation email")

        // Here: SMTP send, external service, etc.
        return nil
    })

    return nil
}

func (p *Plugin) Shutdown() error {
    return nil
}
```

### Example: Minimum Order Value

Before-hooks can **prevent operations** by returning an error:

```go
func (p *Plugin) Init(app *sdk.AppContext) error {
    app.Hooks.On(sdk.HookBeforeCheckout, func(ctx context.Context, event *sdk.HookEvent) error {
        o := event.Entity.(*order.Order)
        if o.Total < 1000 { // prices in cents
            return fmt.Errorf("minimum order value: 10.00 EUR")
        }
        return nil
    })
    return nil
}
```

### Example: Custom API Endpoints

Plugins can register their own endpoints via the Chi router:

```go
func (p *Plugin) Init(app *sdk.AppContext) error {
    app.Router.Route("/api/v1/wishlist", func(r chi.Router) {
        r.Get("/", p.handleList)
        r.Post("/", p.handleAdd)
        r.Delete("/{id}", p.handleRemove)
    })

    return nil
}

func (p *Plugin) handleList(w http.ResponseWriter, r *http.Request) {
    // Direct DB access via p.db (stored during Init)
    rows, err := p.db.Query(r.Context(), "SELECT * FROM wishlists WHERE customer_id = $1", customerID)
    // ...
}
```

### Registering a Plugin

To activate a plugin, register it in `internal/app/app.go` after creating the `App`:

```go
import "github.com/epoxx-arch/stoa/plugins/orderemail"

// In New() or a dedicated method:
func (a *App) RegisterPlugins() error {
    appCtx := &plugin.AppContext{
        DB:     a.DB.Pool,
        Router: a.Server.Router(),
        Config: nil, // or load from config.yaml
        Logger: a.Logger,
    }

    return a.PluginRegistry.Register(orderemail.New(), appCtx)
}
```

### Available Hooks

| Hook | Timing | Can cancel? |
|------|--------|-------------|
| `product.before_create` | Before product creation | Yes |
| `product.after_create` | After product creation | No |
| `product.before_update` | Before product update | Yes |
| `product.after_update` | After product update | No |
| `product.before_delete` | Before product deletion | Yes |
| `product.after_delete` | After product deletion | No |
| `order.before_create` | Before order creation | Yes |
| `order.after_create` | After order creation | No |
| `order.before_update` | Before status change | Yes |
| `order.after_update` | After status change | No |
| `cart.before_add_item` | Before adding to cart | Yes |
| `cart.after_add_item` | After adding to cart | No |
| `cart.before_update_item` | Before quantity change | Yes |
| `cart.after_update_item` | After quantity change | No |
| `cart.before_remove_item` | Before item removal | Yes |
| `cart.after_remove_item` | After item removal | No |
| `customer.before_create` | Before customer registration | Yes |
| `customer.after_create` | After customer registration | No |
| `customer.before_update` | Before customer update | Yes |
| `customer.after_update` | After customer update | No |
| `category.before_create` | Before category creation | Yes |
| `category.after_create` | After category creation | No |
| `category.before_update` | Before category update | Yes |
| `category.after_update` | After category update | No |
| `category.before_delete` | Before category deletion | Yes |
| `category.after_delete` | After category deletion | No |
| `checkout.before` | Before checkout completion | Yes |
| `checkout.after` | After checkout completion | No |
| `payment.after_complete` | After successful payment | No |
| `payment.after_failed` | After failed payment | No |

**Before-hooks** execute before the database operation and can cancel it by returning an error. **After-hooks** execute afterwards -- errors are only logged and do not abort the operation.

---

## Integrating a Payment Service Provider (PSP)

Stoa provides a flexible payment architecture that separates *payment methods* (stored in the database) from *payment processing* (implemented as plugins). This section explains step by step how to integrate a PSP such as Stripe, PayPal, Mollie, or any other provider.

### Architecture Overview

```
┌──────────────┐       ┌──────────────┐       ┌────────────────────┐
│  Storefront  │──────▶│  Stoa API    │──────▶│  PSP Plugin        │
│  (Checkout)  │       │  /checkout   │       │  (e.g. Stripe)     │
└──────────────┘       └──────┬───────┘       └────────┬───────────┘
                              │                        │
                     ┌────────▼────────┐      ┌────────▼───────────┐
                     │ PaymentMethod   │      │ Stripe API         │
                     │ (DB: config,    │      │ (external)         │
                     │  provider name) │      └────────────────────┘
                     └─────────────────┘
```

1. A **PaymentMethod** record in the database stores the provider name (e.g. `"stripe"`) and encrypted provider credentials in the `config` field (e.g. API keys, webhook secrets).
2. A **PSP plugin** listens to checkout/payment hooks, reads the config from the payment method, and communicates with the external provider API.
3. The plugin creates **PaymentTransaction** records to track the outcome.

### Step 1: Create the Plugin Skeleton

Create a new directory for your plugin, e.g. `plugins/stripe/plugin.go`:

```go
package stripe

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/rs/zerolog"

    "github.com/epoxx-arch/stoa/internal/domain/order"
    "github.com/epoxx-arch/stoa/internal/domain/payment"
    "github.com/epoxx-arch/stoa/pkg/sdk"
)

// ProviderName is the identifier stored in payment_methods.provider.
const ProviderName = "stripe"

// Config holds the provider-specific credentials stored (encrypted) in
// PaymentMethod.Config.
type Config struct {
    SecretKey     string `json:"secret_key"`
    WebhookSecret string `json:"webhook_secret"`
    PublishableKey string `json:"publishable_key"`
}

type Plugin struct {
    db     *pgxpool.Pool
    logger zerolog.Logger
    hooks  *sdk.HookRegistry
}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) Name() string        { return "stripe-payment" }
func (p *Plugin) Version() string     { return "1.0.0" }
func (p *Plugin) Description() string { return "Stripe payment integration" }
func (p *Plugin) Shutdown() error     { return nil }
```

### Step 2: Implement Init -- Hook into the Checkout Flow

In the `Init` method you register hooks and optional webhook endpoints:

```go
func (p *Plugin) Init(app *sdk.AppContext) error {
    p.db = app.DB
    p.logger = app.Logger
    p.hooks = app.Hooks

    // 1. Before checkout: create a payment intent with the provider
    app.Hooks.On(sdk.HookBeforeCheckout, p.handleBeforeCheckout)

    // 2. Register a webhook endpoint for async payment confirmations
    app.Router.Route("/api/v1/payments/stripe", func(r chi.Router) {
        r.Post("/webhook", p.handleWebhook)
    })

    p.logger.Info().Msg("stripe payment plugin initialized")
    return nil
}
```

### Step 3: Load Provider Credentials from the PaymentMethod

When the checkout hook fires, you need to look up the `PaymentMethod` to retrieve the (decrypted) config. The config is automatically decrypted by the repository layer -- your plugin receives plain JSON:

```go
func (p *Plugin) loadConfig(ctx context.Context, methodID uuid.UUID) (*Config, error) {
    // Query the payment method directly from the DB.
    var configBytes []byte
    err := p.db.QueryRow(ctx,
        `SELECT config FROM payment_methods WHERE id = $1`, methodID,
    ).Scan(&configBytes)
    if err != nil {
        return nil, fmt.Errorf("stripe: load config: %w", err)
    }

    var cfg Config
    if err := json.Unmarshal(configBytes, &cfg); err != nil {
        return nil, fmt.Errorf("stripe: unmarshal config: %w", err)
    }
    return &cfg, nil
}
```

> **Note:** If you query the database directly (as above), the config column contains the *encrypted* bytes. To get decrypted config, use the `PaymentMethodService.GetByID()` method instead, which goes through the repository layer where decryption happens automatically. You can access the service by storing a reference during `Init`, or by calling the service from the hook event context.

A cleaner approach is to receive the payment method through the hook event:

```go
func (p *Plugin) handleBeforeCheckout(ctx context.Context, event *sdk.HookEvent) error {
    o := event.Entity.(*order.Order)

    // Use the payment method service (injected or looked up) to get decrypted config
    method, err := p.paymentMethodSvc.GetByID(ctx, o.PaymentMethodID)
    if err != nil {
        return fmt.Errorf("stripe: %w", err)
    }
    if method.Provider != ProviderName {
        return nil // not our provider, skip
    }

    var cfg Config
    if err := json.Unmarshal(method.Config, &cfg); err != nil {
        return fmt.Errorf("stripe: invalid config: %w", err)
    }

    // Now use cfg.SecretKey to call the Stripe API...
    return p.createPaymentIntent(ctx, o, &cfg)
}
```

### Step 4: Communicate with the Provider API

Implement the actual API calls to your PSP. This example uses Stripe's PaymentIntents:

```go
func (p *Plugin) createPaymentIntent(ctx context.Context, o *order.Order, cfg *Config) error {
    // Build the request to the Stripe API
    // POST https://api.stripe.com/v1/payment_intents
    //   amount=<o.Total>
    //   currency=<o.Currency>
    //   metadata[order_id]=<o.ID>

    // Use cfg.SecretKey as the Bearer token
    // Parse the response to get the client_secret

    // Store the provider reference (e.g. pi_xxx) for later reconciliation:
    _, err := p.db.Exec(ctx, `
        INSERT INTO payment_transactions
            (id, order_id, payment_method_id, status, currency, amount, provider_reference, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
        uuid.New(), o.ID, o.PaymentMethodID, "pending", o.Currency, o.Total, stripePaymentIntentID,
    )
    return err
}
```

### Step 5: Handle Webhooks for Asynchronous Confirmation

Most PSPs confirm payments asynchronously via webhooks. Register an endpoint and verify the signature:

```go
func (p *Plugin) handleWebhook(w http.ResponseWriter, r *http.Request) {
    // 1. Read and verify the webhook signature
    //    (use cfg.WebhookSecret from the payment method)
    body, _ := io.ReadAll(r.Body)

    // 2. Parse the event type
    //    e.g. "payment_intent.succeeded" or "payment_intent.payment_failed"

    // 3. Update the transaction status
    _, err := p.db.Exec(r.Context(), `
        UPDATE payment_transactions
        SET status = $1
        WHERE provider_reference = $2`,
        "completed", providerReference,
    )
    if err != nil {
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }

    // 4. Fire the appropriate hook so other plugins can react
    if eventType == "payment_intent.succeeded" {
        _ = p.hooks.Dispatch(r.Context(), &sdk.HookEvent{
            Name:   sdk.HookAfterPaymentComplete,
            Entity: transaction,
        })
    } else {
        _ = p.hooks.Dispatch(r.Context(), &sdk.HookEvent{
            Name:   sdk.HookAfterPaymentFailed,
            Entity: transaction,
        })
    }

    w.WriteHeader(http.StatusOK)
}
```

### Step 6: Configure the Payment Method via Admin API

Create a payment method through the admin API. The `config` field holds your provider credentials -- they will be encrypted at rest automatically:

```bash
curl -X POST http://localhost:8080/api/v1/admin/payment-methods \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{
    "provider": "stripe",
    "active": true,
    "config": {
      "secret_key": "sk_live_...",
      "publishable_key": "pk_live_...",
      "webhook_secret": "whsec_..."
    },
    "translations": [
      {"locale": "en-US", "name": "Credit Card", "description": "Pay with Visa, Mastercard, or Amex"},
      {"locale": "de-DE", "name": "Kreditkarte", "description": "Zahlen Sie mit Visa, Mastercard oder Amex"}
    ]
  }'
```

The `config` object is stored as AES-256-GCM encrypted bytes in the database. It is never exposed through the public store API (the field is tagged `json:"-"`). Only the repository layer decrypts it when a service or plugin requests it internally.

### Step 7: Register the Plugin

Add your plugin to `internal/app/app.go`:

```go
import "github.com/epoxx-arch/stoa/plugins/stripe"

func (a *App) RegisterPlugins() error {
    appCtx := &plugin.AppContext{
        DB:     a.DB.Pool,
        Router: a.Server.Router(),
        Hooks:  a.PluginRegistry.Hooks(),
        Logger: a.Logger,
    }
    return a.PluginRegistry.Register(stripe.New(), appCtx)
}
```

### Summary: PSP Integration Checklist

| Step | What | Where |
|------|------|-------|
| 1 | Create plugin struct implementing `sdk.Plugin` | `plugins/<provider>/plugin.go` |
| 2 | Define a `Config` struct matching your provider's credentials | Same file |
| 3 | Hook into `checkout.before` to initiate payment | `Init()` method |
| 4 | Parse `PaymentMethod.Config` (auto-decrypted JSON) for API keys | Hook handler |
| 5 | Call the provider API to create a payment intent/session | Hook handler |
| 6 | Create a `payment_transactions` record with status `pending` | Hook handler |
| 7 | Register a `/api/v1/payments/<provider>/webhook` endpoint | `Init()` method |
| 8 | Verify webhook signature and update transaction status | Webhook handler |
| 9 | Dispatch `payment.after_complete` or `payment.after_failed` hook | Webhook handler |
| 10 | Register the plugin in `app.go` | `RegisterPlugins()` |
| 11 | Create the payment method via admin API with provider credentials | Admin API / UI |

### Security Notes

- **Config encryption**: All provider credentials in `PaymentMethod.Config` are encrypted with AES-256-GCM at rest. Set `STOA_PAYMENT_ENCRYPTION_KEY` (32-byte key or 64-char hex) before starting the application. Existing plaintext configs are automatically migrated on startup.
- **Never expose secrets**: The `Config` field is tagged `json:"-"` and never included in API responses. Only internal services and plugins can access it.
- **Webhook verification**: Always verify webhook signatures using your provider's SDK or signing secret. Never trust unverified webhook payloads.
- **Scope provider access**: Each payment method has its own isolated config. You can run multiple providers (Stripe + PayPal) or multiple accounts of the same provider simultaneously.

---

## License

Apache 2.0 -- see [LICENSE](LICENSE).

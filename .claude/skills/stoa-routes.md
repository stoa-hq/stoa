# Stoa: API-Routen & MCP Server Referenz

## REST API Routen

```
GET  /api/v1/health                          → kein Auth

POST /api/v1/auth/login                      → kein Auth
POST /api/v1/auth/refresh                    → kein Auth
POST /api/v1/auth/logout                     → kein Auth

# Admin (JWT required, role: super_admin|admin|manager)
GET|POST        /api/v1/admin/products
GET|PUT|DELETE  /api/v1/admin/products/{id}
POST            /api/v1/admin/products/{id}/variants
GET|POST        /api/v1/admin/categories
GET|PUT|DELETE  /api/v1/admin/categories/{id}
GET|POST        /api/v1/admin/customers
GET|PUT|DELETE  /api/v1/admin/customers/{id}
GET             /api/v1/admin/orders
GET             /api/v1/admin/orders/{id}
PUT             /api/v1/admin/orders/{id}/status
GET|POST        /api/v1/admin/tax-rules
GET|PUT|DELETE  /api/v1/admin/tax-rules/{id}
GET|POST        /api/v1/admin/shipping-methods
GET|PUT|DELETE  /api/v1/admin/shipping-methods/{id}
GET|POST        /api/v1/admin/payment-methods
GET|PUT|DELETE  /api/v1/admin/payment-methods/{id}
GET|POST        /api/v1/admin/discounts
GET|PUT|DELETE  /api/v1/admin/discounts/{id}
POST            /api/v1/admin/discounts/validate
POST            /api/v1/admin/discounts/{id}/apply
GET|POST        /api/v1/admin/tags
GET|PUT|DELETE  /api/v1/admin/tags/{id}
GET             /api/v1/admin/audit-log
GET|POST        /api/v1/admin/media
GET|DELETE      /api/v1/admin/media/{id}
POST            /api/v1/admin/api-keys
GET             /api/v1/admin/api-keys
DELETE          /api/v1/admin/api-keys/{id}

# Store (OptionalAuth)
GET             /api/v1/store/products
GET             /api/v1/store/products/{slug}
GET             /api/v1/store/categories/tree
POST            /api/v1/store/register
GET|PUT         /api/v1/store/account
POST            /api/v1/store/checkout
GET             /api/v1/store/account/orders
POST            /api/v1/store/cart
GET             /api/v1/store/cart/{id}
POST            /api/v1/store/cart/{id}/items
PUT             /api/v1/store/cart/{id}/items/{itemId}
DELETE          /api/v1/store/cart/{id}/items/{itemId}
GET             /api/v1/store/shipping-methods
GET             /api/v1/store/shipping-methods/{id}
GET             /api/v1/store/payment-methods
GET             /api/v1/store/payment-methods/{id}
GET             /api/v1/store/search?q=

# SPAs
/admin    → Admin SPA (internal/admin/build/, embed.FS)
/admin/*  → Admin SPA catch-all
/         → Storefront SPA (internal/storefront/build/, embed.FS)
/*        → Storefront catch-all
```

## MCP Server

Zwei separate MCP Server für KI-Agenten-Interaktion.

### Architektur

```
internal/mcp/
  config.go        → Env-Config (STOA_MCP_API_URL, STOA_MCP_API_KEY, STOA_MCP_TRANSPORT)
  client.go        → StoaClient HTTP-Wrapper mit API-Key Auth
  errors.go        → API-Fehler → MCP CallToolResult Mapping
  response.go      → Stoa Response-Envelope Parsing + FormatResponse()
  store/tools.go   → RegisterTools() — 16 Store-Tools
  admin/tools.go   → RegisterTools() — 33 Admin-Tools
```

### Store MCP Server (16 Tools)

- **Produkte**: `store_list_products`, `store_get_product`, `store_search`, `store_get_categories`
- **Warenkorb**: `store_create_cart`, `store_get_cart`, `store_add_to_cart`, `store_update_cart_item`, `store_remove_from_cart`
- **Checkout**: `store_get_shipping_methods`, `store_get_payment_methods`, `store_checkout`
- **Account**: `store_register`, `store_login`, `store_get_account`, `store_list_orders`

### Admin MCP Server (33 Tools)

- **Produkte** (8): CRUD + Varianten
- **Bestellungen** (3): Listen, Details, Status-Update
- **Rabatte** (5): CRUD + Validate + Apply
- **Kunden** (4): CRUD
- **Kategorien** (4): CRUD
- **Tags** (3): Erstellen, Listen, Löschen
- **Media** (2): Listen, Löschen
- **Config** (3): Versand-, Steuer-, Zahlungsmethoden auflisten
- **Audit** (1): Audit-Log auflisten

### Konfiguration

Env-Variablen:
- `STOA_MCP_API_URL` — Backend-URL (default: `http://localhost:8080`)
- `STOA_MCP_API_KEY` — API-Key für Authentifizierung
- `STOA_MCP_TRANSPORT` — `stdio` (default) oder `http`
- `STOA_MCP_HTTP_PORT` — HTTP-Port wenn Transport=http (default: 8090)

Build & Run:
```bash
make mcp-store-build   # Binary → bin/stoa-store-mcp
make mcp-admin-build   # Binary → bin/stoa-admin-mcp
make mcp-store-run     # Build + run (stdio)
make mcp-admin-run     # Build + run (stdio)
```

### Claude Code Integration

```json
{
  "mcpServers": {
    "stoa-store": {
      "command": "/pfad/zu/stoa-store-mcp",
      "env": {
        "STOA_MCP_API_URL": "http://localhost:8080",
        "STOA_MCP_API_KEY": "ck_..."
      }
    },
    "stoa-admin": {
      "command": "/pfad/zu/stoa-admin-mcp",
      "env": {
        "STOA_MCP_API_URL": "http://localhost:8080",
        "STOA_MCP_API_KEY": "ck_..."
      }
    }
  }
}
```

### API-Key Permissions

API-Keys über `/api/v1/admin/api-keys` (nur `super_admin`/`admin`):
- `RequireRole` lässt `api_client` mit passenden Permissions durch
- Feinkörnige Kontrolle via `RequirePermission`
- Permission Format: `{entity}.{action}` (z.B. `products.create`)

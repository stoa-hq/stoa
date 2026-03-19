# Stoa: API-Routen & MCP Server Referenz

## REST API Routen

```
GET  /api/v1/health                                        → kein Auth

POST /api/v1/auth/login                                    → kein Auth
POST /api/v1/auth/refresh                                  → kein Auth
POST /api/v1/auth/logout                                   → kein Auth

# Admin (JWT required, role: super_admin|admin|manager)
GET|POST        /api/v1/admin/products
POST            /api/v1/admin/products/bulk
POST            /api/v1/admin/products/import
GET             /api/v1/admin/products/import/template
GET|PUT|DELETE  /api/v1/admin/products/{id}
POST            /api/v1/admin/products/{id}/variants
PUT|DELETE      /api/v1/admin/products/{id}/variants/{variantId}
GET             /api/v1/admin/products/{productID}/stock
GET|POST        /api/v1/admin/categories
GET|PUT|DELETE  /api/v1/admin/categories/{id}
GET|POST        /api/v1/admin/customers
GET|PUT|DELETE  /api/v1/admin/customers/{id}
GET             /api/v1/admin/orders
GET             /api/v1/admin/orders/{id}
PUT             /api/v1/admin/orders/{id}/status
GET             /api/v1/admin/orders/{orderID}/transactions
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
GET|POST        /api/v1/admin/warehouses
GET|PUT|DELETE  /api/v1/admin/warehouses/{id}
GET|PUT         /api/v1/admin/warehouses/{id}/stock
DELETE          /api/v1/admin/warehouses/{id}/stock/{stockID}
GET|POST        /api/v1/admin/property-groups
GET|PUT|DELETE  /api/v1/admin/property-groups/{id}
POST            /api/v1/admin/property-groups/{id}/options
PUT|DELETE      /api/v1/admin/property-groups/{id}/options/{optId}
GET|PUT         /api/v1/admin/settings
GET             /api/v1/admin/config
POST            /api/v1/admin/api-keys
GET             /api/v1/admin/api-keys
DELETE          /api/v1/admin/api-keys/{id}
GET             /api/v1/admin/plugin-manifest

# Store (OptionalAuth)
GET             /api/v1/store/products
GET             /api/v1/store/products/id/{id}
GET             /api/v1/store/products/{slug}
GET             /api/v1/store/categories/tree
POST            /api/v1/store/register
GET|PUT         /api/v1/store/account
POST            /api/v1/store/checkout
GET             /api/v1/store/account/orders
GET             /api/v1/store/orders/{orderID}/transactions
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
GET             /api/v1/store/settings
GET             /api/v1/store/config
GET             /api/v1/store/plugin-manifest

# Static & SPAs
/uploads/*                    → Uploaded media files (local storage)
/plugins/{name}/assets/*      → Per-plugin embedded static files
/admin, /admin/*              → Admin SPA (embed.FS, dynamic CSP)
/, /*                         → Storefront SPA (embed.FS, dynamic CSP)
```

## MCP Server

Zwei separate MCP Server für KI-Agenten-Interaktion.

### Architektur

```
cmd/stoa-store-mcp/     → Store MCP Binary (SSE, plugin tool registration)
cmd/stoa-admin-mcp/     → Admin MCP Binary
internal/mcp/
  config.go             → Env-Config (STOA_MCP_API_URL, STOA_MCP_API_KEY, STOA_MCP_TRANSPORT)
  client.go             → StoaClient HTTP-Wrapper mit API-Key Auth
  errors.go             → API-Fehler → MCP CallToolResult Mapping
  response.go           → Stoa Response-Envelope Parsing + FormatResponse()
  scoped.go             → ScopedMCPServer — Plugin Tool-Prefix Enforcement
  store_client.go       → StoreScopedClient — Plugin Client auf /api/v1/store/* beschränkt
  store/tools.go        → RegisterTools() — 16 Store-Tools
  admin/tools.go        → RegisterTools() — 41 Admin-Tools
```

### Store MCP Server (16 Tools)

| Gruppe | Tools |
|--------|-------|
| Produkte | `store_list_products`, `store_get_product`, `store_search`, `store_get_categories` |
| Warenkorb | `store_create_cart`, `store_get_cart`, `store_add_to_cart`, `store_update_cart_item`, `store_remove_from_cart` |
| Checkout | `store_get_shipping_methods`, `store_get_payment_methods`, `store_checkout` |
| Account | `store_register`, `store_login`, `store_get_account`, `store_list_orders` |

### Admin MCP Server (41 Tools)

| Gruppe | Tools |
|--------|-------|
| Produkte (8) | `admin_list_products`, `admin_get_product`, `admin_create_product`, `admin_update_product`, `admin_delete_product`, `admin_create_variant`, `admin_update_variant`, `admin_delete_variant` |
| Bestellungen (3) | `admin_list_orders`, `admin_get_order`, `admin_update_order_status` |
| Rabatte (5) | `admin_list_discounts`, `admin_get_discount`, `admin_create_discount`, `admin_update_discount`, `admin_delete_discount` |
| Kunden (4) | `admin_list_customers`, `admin_get_customer`, `admin_update_customer`, `admin_delete_customer` |
| Kategorien (4) | `admin_list_categories`, `admin_get_category`, `admin_create_category`, `admin_update_category` |
| Tags (3) | `admin_list_tags`, `admin_create_tag`, `admin_delete_tag` |
| Media (2) | `admin_list_media`, `admin_delete_media` |
| Warehouses (8) | `admin_list_warehouses`, `admin_get_warehouse`, `admin_create_warehouse`, `admin_update_warehouse`, `admin_delete_warehouse`, `admin_get_warehouse_stock`, `admin_set_warehouse_stock`, `admin_get_product_stock` |
| Config (3) | `admin_list_shipping_methods`, `admin_list_tax_rules`, `admin_list_payment_methods` |
| Audit (1) | `admin_list_audit_log` |

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

API-Keys via `/api/v1/admin/api-keys` (nur `super_admin`/`admin`):
- `RequireRole` lässt `api_client` mit Permissions durch
- Feinkörnige Kontrolle via `RequirePermission`
- Format: `{entity}.{action}` (z.B. `products.create`)

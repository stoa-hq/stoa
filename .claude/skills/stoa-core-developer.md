# Stoa Core Developer Skill

Expert-level knowledge for maintaining and extending Stoa's internal Go architecture. Covers all layers: domain packages, HTTP server, auth, MCP, plugin system, and CLI.

## Architecture

```
cmd/stoa/                      → Cobra CLI (serve, migrate, admin, seed, plugin, version)
cmd/stoa-store-mcp/            → Store MCP Server (SSE, plugin tool registration)
cmd/stoa-admin-mcp/            → Admin MCP Server
internal/app/app.go            → DI container — wires repos/services/handlers/routes/plugins
internal/server/server.go      → HTTP server + middleware chain (chi)
internal/auth/                 → JWT, API keys, RBAC, CSRF, Argon2id, brute-force, token blacklist/store
internal/domain/<entity>/      → 14 domain packages (6-file pattern)
internal/mcp/                  → MCP infra: client, scoped server, store-scoped client
internal/plugin/               → Plugin registry, installer, manifest handler, code generation
internal/search/               → PostgreSQL full-text search
internal/media/                → Local/S3 storage + image processor
internal/settings/             → Store settings (domain-like, 6 files)
internal/crypto/               → AES encryption (payment data)
internal/csp/                  → Content Security Policy management
internal/config/               → Viper config
pkg/sdk/                       → Public plugin SDK (Plugin, UIPlugin, MCPStorePlugin, HookRegistry)
migrations/                    → 10 golang-migrate SQL migrations
admin/                         → SvelteKit 5 Admin SPA
storefront/                    → SvelteKit 5 Storefront SPA
```

**14 Domains**: product, category, customer, order, cart, tax, shipping, payment, discount, tag, audit, media, warehouse, settings

## Domain Package Pattern (6 Files)

Reference implementation: `internal/domain/product/`. Product also has extra `bulk.go` for bulk import. Audit has `middleware.go` instead of standard pattern.

### 1. `entity.go` — Domain Model

Rules:
- Prices as **integer cents** (1999 = 19.99 EUR), tax rates as **basis points** (1900 = 19%)
- All entities: `CustomFields map[string]interface{}` (user-facing) + `Metadata map[string]interface{}` (internal) as JSONB
- Translations: separate structs with `(entity_id, locale)` composite key, lazy-loaded
- UUIDs from `github.com/google/uuid`, timestamps always UTC
- Define `var ErrNotFound = errors.New("not found")` per package

### 2. `repository.go` — Interface

Rules:
- Contract-first: interface defines persistence API
- `FindAll` returns `(items, totalCount, error)` for pagination
- Domain `ErrNotFound` for missing entities
- Context-first parameters always

### 3. `postgres.go` — Implementation

Rules:
- **pgxpool** for connection pooling
- **Parameterized queries** — `$1, $2, ...` only, never string interpolation
- **Sort column allowlist** — map of valid column names (SQL injection prevention)
- **Pagination**: validate page >= 1, limit in [1..100], offset = (page-1)*limit
- **N+1 prevention**: `WHERE entity_id = ANY($1)` for batch-loading relations
- **Transactions**: `tx, err := r.db.Begin(ctx)` / `defer tx.Rollback(ctx)` / `tx.Commit(ctx)`
- **JSONB**: `marshalJSON`/`unmarshalJSON` helper functions
- **UUID**: `if p.ID == uuid.Nil { p.ID = uuid.New() }`
- **Timestamps**: `time.Now().UTC()` on create/update
- **Error mapping**: `pgx.ErrNoRows` → `ErrNotFound`

### 4. `service.go` — Business Logic

Rules:
- Before-hooks can cancel operations (return error → propagated to caller)
- After-hooks are best-effort (error logged, operation not rolled back)
- For updates, fetch existing entity first; pass as `Entity` with `Changes` map in HookEvent
- DI via constructor — function fields for optional deps (`mediaURLFn`, `taxRateFn`)

Hook dispatch pattern:
```go
// Before-hook: abort on error
if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
    Name: sdk.HookBeforeProductCreate, Entity: p,
}); err != nil {
    return fmt.Errorf("hook %s: %w", sdk.HookBeforeProductCreate, err)
}
// ... do work ...
// After-hook: log only
if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
    Name: sdk.HookAfterProductCreate, Entity: p,
}); err != nil {
    s.logger.Warn().Err(err).Str("hook", sdk.HookAfterProductCreate).Msg("after-hook error")
}
```

### 5. `handler.go` — HTTP Endpoints

Rules:
- **Local response helpers** per handler — NOT shared across packages: `apiResponse`, `apiMeta`, `apiError`, `writeJSON`, `writeError`, `serverError`
- **Two route groups**: `RegisterAdminRoutes(r chi.Router)` + `RegisterStoreRoutes(r chi.Router)`
- **Handler flow**: decode → validate → process → respond
- Chi URL params: `chi.URLParam(r, "id")` with `parseUUID` helper
- Pagination defaults: page=1, limit=25 (max 100)
- Locale from `Accept-Language` header, default "en"
- Never leak internal errors — log full, return generic

### 6. `dto.go` — Request/Response Mapping

Rules:
- Create DTOs: required fields as **values**
- Update DTOs: optional fields as **pointers** (nil = "not sent")
- go-playground/validator tags for validation
- Separate conversion functions: `FromCreateRequest`, `ApplyUpdateRequest`, `ToResponse`, `ToResponseList`

## HTTP Server & Middleware

Middleware chain (order, `internal/server/server.go`):
1. Recoverer → 2. Request ID → 3. Structured logging → 4. CORS → 5. Security headers → 6. Rate limiter → 7. CSRF (exempt with `Authorization`) → 8. Content-Type enforcement

Route groups (`internal/app/app.go`):
- `/api/v1/admin`: `Authenticate` + `RequireRole(SuperAdmin, Admin, Manager)` + domain admin routes
- `/api/v1/store`: `OptionalAuth` + audit middleware + domain store routes

## Auth System (`internal/auth/`)

Context helpers: `auth.UserID(ctx)`, `auth.UserType(ctx)`, `auth.UserRole(ctx)`, `auth.UserPermissions(ctx)`

Middleware: `Authenticate` (requires valid token), `OptionalAuth` (extract if present), `RequireRole(roles...)`, `RequirePermission(perm)`

Context keys: `type contextKeyType string` in `middleware.go`. SDK mirrors via `AuthHelper` function fields.

Files: `jwt.go`, `middleware.go`, `handler.go`, `apikey.go`, `password.go`, `permissions.go`, `bruteforce.go`, `blacklist.go`, `token_store.go`

## Plugin System Internals

### SDK (`pkg/sdk/`)

```
plugin.go    → Plugin interface, AppContext (DB, Router, AssetRouter, Hooks, Config, Logger, Auth)
registry.go  → Global Register/RegisteredPlugins (sync.Mutex)
hooks.go     → HookRegistry (sync.RWMutex), HookEvent, 29+ hook constants
mcp.go       → MCPStorePlugin, StoreAPIClient interfaces
ui.go        → UIPlugin, UIExtension, UISchema, UIComponent, ValidateUIExtension()
entities.go  → BaseEntity shared fields
```

**AppContext**: Plugins get ROOT chi router (NOT store/admin subrouter). Must apply auth middleware explicitly.

### Plugin Registry (`internal/plugin/`)

```
registry.go          → Registry.Register/ShutdownAll/CollectUIExtensions/UIExtensions
manifest_handler.go  → ManifestHandler: GET /api/v1/{store,admin}/plugin-manifest
installer.go         → KnownPlugins map, Install/Remove/ListInstalled, code generation
interfaces.go        → Type aliases for sdk types
```

Installer workflow: `ResolvePackage` → `ensurePluginsModFile` (go.plugins.mod) → `go get` → `writePluginsFile` (generates `plugins_generated.go` in both `cmd/stoa/` and `cmd/stoa-store-mcp/`) → `rebuild`

### MCP Plugin Isolation (`internal/mcp/`)

1. **ScopedMCPServer** (`scoped.go`): enforces tool prefix `store_{pluginName}_*`
2. **StoreScopedClient** (`store_client.go`): restricts to `/api/v1/store/*`, blocks `..`
3. **Panic recovery** (`cmd/stoa-store-mcp/main.go`): `safeRegisterPluginTools` wraps in `recover()`

## DI Container (`internal/app/app.go`)

Wiring order in `setupDomains`:
1. Create repositories (take `*pgxpool.Pool`)
2. Create services (take repos + hooks + logger + optional function deps)
3. Create handlers (take services + validator + logger)
4. Mount routes on admin/store router groups

## Database & Migrations

PostgreSQL 16+, pgxpool, golang-migrate. Extensions: `uuid-ossp`, `pg_trgm`.
10 migrations: init, cart_item_upsert, shipping_tax_rule, payment_tx_unique_provider_ref, order_guest_token, order_payment_reference, warehouse, warehouse_negative_stock, store_settings, refresh_token_store.
Naming: `000NNN_description.{up,down}.sql`. Always provide both up and down.

## Error Handling

```
Repository:  pgx.ErrNoRows → ErrNotFound; else fmt.Errorf("findByID: %w", err)
Service:     errors.Is(err, ErrNotFound) → propagate; else fmt.Errorf("service X: %w", err)
Handler:     errors.Is(err, ErrNotFound) → notFound(w, "not found"); else serverError(w, r, err)
```

Never expose internal errors. Log full error, return generic message.

## Security Checklist

1. **SQL injection**: parameterized queries, allowlisted sort columns
2. **Auth on routes**: store = `OptionalAuth`, admin = `Authenticate` + `RequireRole`
3. **IDOR prevention**: store endpoints filter by `customer_id` from auth context; guests by `guest_token`
4. **Error sanitization**: no DB errors, internal paths, or stack traces in API responses
5. **Body size limits**: `io.LimitReader` for user-submitted bodies
6. **CSRF**: Double Submit Cookie — exempt for Bearer/ApiKey auth
7. **Webhooks**: verify signatures, background context for goroutines, idempotency via `ON CONFLICT DO NOTHING`
8. **Plugin isolation**: ScopedMCPServer (tool prefix), StoreScopedClient (path restriction), panic recovery
9. **Context propagation**: always pass `ctx`; `context.Background()` with timeout for detached goroutines
10. **Plugin UI**: tag prefix `stoa-{pluginName}-`, no `..` in URLs, Light DOM + scoped CSS, SRI, dynamic CSP

## Cross-references

- New domain → `/stoa-new-domain`
- New plugin → `/stoa-plugin-developer`
- Test patterns → `/stoa-test`
- API routes + MCP → `/stoa-routes`

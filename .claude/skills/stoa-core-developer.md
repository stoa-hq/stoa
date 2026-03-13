# Stoa Core Developer Skill

You are an expert Stoa core developer. You help maintain and extend the internal architecture of the Stoa e-commerce platform. You understand all layers: domain packages, HTTP server, auth, MCP infrastructure, plugin system, and CLI.

## Architecture Overview

```
cmd/stoa/                      → Cobra CLI (serve, migrate, seed, plugin, admin)
cmd/stoa-store-mcp/            → Store MCP Server (SSE, plugin tool registration)
cmd/stoa-admin-mcp/            → Admin MCP Server
internal/app/app.go            → DI container — wires all repos/services/handlers/routes/plugins
internal/server/server.go      → HTTP server + middleware chain (chi)
internal/auth/                 → JWT, API keys, RBAC, CSRF, Argon2id
internal/domain/<entity>/      → 12 domain packages (6-file pattern)
internal/mcp/                  → MCP infra: client, scoped server, store-scoped client
internal/plugin/               → Plugin registry, installer, code generation
internal/search/               → PostgreSQL full-text search
internal/media/                → Local/S3 storage + image processor
internal/settings/             → Read-only config endpoint
internal/config/               → Viper config
pkg/sdk/                       → Public plugin SDK (Plugin, AppContext, AuthHelper, hooks, MCP interfaces)
migrations/                    → golang-migrate SQL migrations
admin/                         → SvelteKit 5 Admin SPA
storefront/                    → SvelteKit 5 Storefront SPA
```

## Domain Package Pattern (6 Files)

Every domain follows this exact structure. Use `internal/domain/product/` as the reference implementation.

### 1. `entity.go` — Domain Model

```go
package product

type Product struct {
    ID           uuid.UUID
    SKU          string
    Active       bool
    PriceNet     int                    // Cents (1999 = €19.99)
    PriceGross   int                    // Cents
    Currency     string                 // ISO 4217 ("EUR")
    TaxRuleID    *uuid.UUID
    Stock        int
    CustomFields map[string]interface{} // User-facing JSONB
    Metadata     map[string]interface{} // Internal JSONB
    Translations []ProductTranslation   // Lazy-loaded
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

var ErrNotFound = errors.New("not found")
```

Rules:
- Prices as **integer cents** (1999 = €19.99), tax rates as **basis points** (1900 = 19%)
- All entities have `CustomFields` (user-facing) + `Metadata` (internal) as JSONB
- Translations are separate structs with `(entity_id, locale)` composite key
- Relations (translations, categories, media) are populated lazily, not in core struct by default
- UUIDs from `github.com/google/uuid`, timestamps always UTC

### 2. `repository.go` — Interface

```go
type ProductRepository interface {
    FindByID(ctx context.Context, id uuid.UUID) (*Product, error)
    FindAll(ctx context.Context, filter ProductFilter) ([]Product, int, error)
    Create(ctx context.Context, p *Product) error
    Update(ctx context.Context, p *Product) error
    Delete(ctx context.Context, id uuid.UUID) error
}

type ProductFilter struct {
    Page       int
    Limit      int
    Active     *bool
    CategoryID *uuid.UUID
    Search     string
    Sort       string  // Allowlisted column name
    Order      string  // "asc" or "desc"
}
```

Rules:
- Contract-first: interface defines persistence API
- `FindAll` returns `(items, totalCount, error)` — total for pagination
- Domain error `ErrNotFound` for missing entities
- Context-first parameters always

### 3. `postgres.go` — Implementation

Key patterns:
- **pgxpool** (`github.com/jackc/pgx/v5/pgxpool`) for connection pooling
- **Parameterized queries** — all values via `$1, $2, ...`, never string interpolation
- **Sort column allowlist** — map of valid column names to prevent SQL injection
- **Pagination**: validate page >= 1, limit in [1..100], offset = (page-1)*limit
- **N+1 prevention**: batch-load relations with `WHERE entity_id = ANY($1)`
- **Transactions** for multi-table operations:

```go
tx, err := r.db.Begin(ctx)
if err != nil {
    return fmt.Errorf("begin tx: %w", err)
}
defer tx.Rollback(ctx) //nolint:errcheck
// ... operations on tx ...
return tx.Commit(ctx)
```

- **JSONB handling**: `marshalJSON`/`unmarshalJSON` helper functions
- **UUID generation**: `if p.ID == uuid.Nil { p.ID = uuid.New() }`
- **Timestamps**: `time.Now().UTC()` on create/update
- **Error mapping**: `pgx.ErrNoRows` → `ErrNotFound`

### 4. `service.go` — Business Logic

```go
type Service struct {
    repo   ProductRepository
    hooks  *sdk.HookRegistry
    logger zerolog.Logger
}

func (s *Service) Create(ctx context.Context, p *Product) error {
    // Before-hook: can abort
    if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
        Name:   sdk.HookBeforeProductCreate,
        Entity: p,
    }); err != nil {
        return fmt.Errorf("hook %s: %w", sdk.HookBeforeProductCreate, err)
    }

    if err := s.repo.Create(ctx, p); err != nil {
        return fmt.Errorf("service Create: %w", err)
    }

    // After-hook: errors logged, don't fail
    if err := s.hooks.Dispatch(ctx, &sdk.HookEvent{
        Name:   sdk.HookAfterProductCreate,
        Entity: p,
    }); err != nil {
        s.logger.Warn().Err(err).Str("hook", sdk.HookAfterProductCreate).Msg("after-hook error")
    }
    return nil
}
```

Rules:
- Before-hooks can cancel operations (return error → propagated to caller)
- After-hooks are best-effort (error logged, operation not rolled back)
- For updates, fetch existing entity first and pass as `Entity` with `Changes` map
- Dependency injection via constructor — function fields for optional dependencies (`mediaURLFn`, `taxRateFn`)

### 5. `handler.go` — HTTP Endpoints

```go
type Handler struct {
    service   *Service
    validator *validator.Validate
    logger    zerolog.Logger
}
```

**Local response helpers** (per handler — NOT shared across packages):

```go
type apiResponse struct {
    Data   interface{} `json:"data,omitempty"`
    Meta   *apiMeta    `json:"meta,omitempty"`
    Errors []apiError  `json:"errors,omitempty"`
}
type apiMeta struct {
    Total int `json:"total"`
    Page  int `json:"page"`
    Limit int `json:"limit"`
    Pages int `json:"pages"`
}
type apiError struct {
    Code   string `json:"code"`
    Detail string `json:"detail"`
    Field  string `json:"field,omitempty"`
}
```

**Two route groups** — admin and store:

```go
func (h *Handler) RegisterAdminRoutes(r chi.Router) {
    r.Get("/products", h.adminList)
    r.Post("/products", h.adminCreate)
    r.Put("/products/{id}", h.adminUpdate)
    r.Delete("/products/{id}", h.adminDelete)
}

func (h *Handler) RegisterStoreRoutes(r chi.Router) {
    r.Get("/products", h.storeList)        // Active only
    r.Get("/products/id/{id}", h.storeGetByID)
    r.Get("/products/{slug}", h.storeGetBySlug)
}
```

**Handler flow**: decode → validate → process → respond

```go
func (h *Handler) adminCreate(w http.ResponseWriter, r *http.Request) {
    var req CreateProductRequest
    if !h.decodeJSON(w, r, &req) { return }
    if !h.validate(w, &req) { return }
    p := FromCreateRequest(&req)
    if err := h.service.Create(r.Context(), p); err != nil {
        h.serverError(w, r, err)
        return
    }
    h.writeJSON(w, http.StatusCreated, apiResponse{Data: ToResponse(p)})
}
```

Rules:
- Each handler package defines its own `writeJSON`, `writeError`, `serverError` helpers
- Chi URL params via `chi.URLParam(r, "id")` — use `parseUUID` helper
- Pagination: defaults page=1, limit=25 (max 100)
- Locale from `Accept-Language` header, default "en"
- Never leak internal errors — log full error, return generic message

### 6. `dto.go` — Request/Response Mapping

```go
// Create: all required fields
type CreateProductRequest struct {
    SKU      string `json:"sku"       validate:"max=100"`
    PriceNet int    `json:"price_net" validate:"min=0"`
    Currency string `json:"currency"  validate:"required,len=3"`
    // ...
}

// Update: pointer fields (nil = not changed, zero value = set to zero)
type UpdateProductRequest struct {
    SKU      *string `json:"sku"       validate:"omitempty,max=100"`
    PriceNet *int    `json:"price_net" validate:"omitempty,min=0"`
    // ...
}

// Conversion functions
func FromCreateRequest(req *CreateProductRequest) *Product { ... }
func ApplyUpdateRequest(p *Product, req *UpdateProductRequest) { ... }
func ToResponse(p *Product) ProductResponse { ... }
func ToResponseList(products []Product) ProductListResponse { ... }
```

Rules:
- Create DTOs: required fields as values
- Update DTOs: optional fields as **pointers** — nil means "not sent"
- go-playground/validator tags for validation
- Separate conversion functions (not methods on DTO)

## HTTP Server & Middleware Stack

**Middleware chain** (in order, defined in `internal/server/server.go`):

1. `chimw.Recoverer` — panic → 500
2. Request ID — `X-Request-ID` header (generated or accepted)
3. Structured logging — method, path, status, duration, request_id
4. CORS — configured origins/headers
5. Security headers — `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, etc.
6. Rate limiter — `httprate.LimitByIP`
7. CSRF — Double Submit Cookie (exempt if `Authorization` header present)
8. Content-Type enforcement — POST/PUT/PATCH require `application/json` or `multipart/form-data`

**Route groups** (in `internal/app/app.go`):

```go
r.Route("/api/v1/admin", func(r chi.Router) {
    r.Use(a.AuthMiddleware.Authenticate)
    r.Use(a.AuthMiddleware.RequireRole(auth.RoleSuperAdmin, auth.RoleAdmin, auth.RoleManager))
    // ... admin domain routes
})

r.Route("/api/v1/store", func(r chi.Router) {
    r.Use(a.AuthMiddleware.OptionalAuth)
    r.Use(audit.Middleware(auditSvc, log))
    // ... store domain routes
})
```

## Auth System (`internal/auth/`)

**Context helpers** (used throughout the codebase):

```go
auth.UserID(ctx)          // uuid.UUID — uuid.Nil if anonymous
auth.UserType(ctx)        // "admin", "customer", "api_key", or ""
auth.UserRole(ctx)        // Role type — "super_admin", "admin", "manager", "customer", "api_client"
auth.UserPermissions(ctx) // []Permission — for API key fine-grained access
```

**Middleware types**:
- `Authenticate` — requires valid token, returns 401 otherwise
- `OptionalAuth` — extracts auth if present, never blocks
- `RequireRole(roles...)` — checks user has one of the allowed roles
- `RequirePermission(perm)` — checks fine-grained permission

**Context keys**: defined as `type contextKeyType string` in `internal/auth/middleware.go`. The SDK mirrors these via `AuthHelper` function fields so plugins don't import internal packages.

## Plugin System Internals

### SDK (`pkg/sdk/`)

Public API for plugins:

```
pkg/sdk/
├── plugin.go    → Plugin interface, AppContext (incl. AssetRouter), AuthHelper
├── registry.go  → Global Register/RegisteredPlugins (sync.Mutex)
├── hooks.go     → HookRegistry (sync.RWMutex), HookEvent, hook constants
├── mcp.go       → MCPStorePlugin, StoreAPIClient interfaces
├── ui.go        → UIPlugin, UIExtension, UISchema, UIComponent, ValidateUIExtension()
├── entities.go  → BaseEntity shared fields
```

**AppContext wiring** (in `internal/app/app.go`):

```go
// Per-plugin AppContext with dedicated asset router
pluginAppCtx := &sdk.AppContext{
    DB:          db.Pool,
    Router:      srv.Router(),      // ROOT chi router
    AssetRouter: assetRouter,       // Mounted at /plugins/{name}/assets/
    Config:      cfg.Plugins,
    Logger:      logger,
    Auth: &sdk.AuthHelper{
        OptionalAuth: authMiddleware.OptionalAuth,
        Required:     authMiddleware.Authenticate,
        UserID:       auth.UserID,
        UserType:     auth.UserType,
    },
}
```

**IMPORTANT**: Plugins get the ROOT router, not a subrouter. The store/admin middleware groups are NOT inherited. Plugins must apply auth middleware explicitly.

### Plugin Registry (`internal/plugin/`)

```
internal/plugin/
├── registry.go          → Registry.Register/ShutdownAll/CollectUIExtensions/UIExtensions
├── manifest_handler.go  → ManifestHandler: GET /api/v1/{store,admin}/plugin-manifest
├── installer.go         → KnownPlugins, Install/Remove/ListInstalled, code generation
├── interfaces.go        → Type aliases: Plugin = sdk.Plugin, AppContext = sdk.AppContext, etc.
```

**UI Extension Collection**: After all plugins are registered, `CollectUIExtensions()` iterates over plugins implementing `sdk.UIPlugin`, validates their extensions, and caches them. `ManifestHandler` serves them filtered by slot prefix (`storefront:*` or `admin:*`).

**Installer workflow**:
1. `ResolvePackage` — short name → import path (via `KnownPlugins` map)
2. `ensurePluginsModFile` — creates isolated `go.plugins.mod`
3. `go get -modfile=go.plugins.mod pkg@latest`
4. `writePluginsFile` — generates **both** `cmd/stoa/plugins_generated.go` and `cmd/stoa-store-mcp/plugins_generated.go`
5. `rebuild` — `go build -modfile=go.plugins.mod`

**Adding a new known plugin**: add entry to `KnownPlugins` map in `installer.go`.

### MCP Plugin Isolation (`internal/mcp/`)

Three security layers:

1. **ScopedMCPServer** (`scoped.go`): wraps `*server.MCPServer`, enforces tool name prefix `store_{pluginName}_*`. Panics on violation (caught by `safeRegisterPluginTools` in main.go).

2. **StoreScopedClient** (`store_client.go`): wraps `StoaClient`, only allows `/api/v1/store/*` paths. Blocks path traversal (`..`).

3. **Panic recovery** (`cmd/stoa-store-mcp/main.go`): `safeRegisterPluginTools` wraps `RegisterStoreMCPTools` in `defer recover()`. Buggy plugin is skipped, server continues.

## DI Container (`internal/app/app.go`)

**Wiring order** in `setupDomains`:

1. Create repositories (all take `*pgxpool.Pool`)
2. Create services (take repos + hooks + logger + optional function deps)
3. Create handlers (take services + validator + logger)
4. Mount routes on router groups

```go
func (a *App) setupDomains(cfg *config.Config) error {
    pool := a.DB.Pool
    hooks := a.PluginRegistry.Hooks()

    // Repos
    productRepo := product.NewPostgresRepository(pool)
    categoryRepo := category.NewPostgresRepository(pool)
    // ...

    // Services
    productSvc := product.NewService(productRepo, hooks, log, mediaURLFn, taxRateFn)
    // ...

    // Handlers
    productH := product.NewHandler(productSvc, validate, log)
    // ...

    // Routes
    r.Route("/api/v1/admin", func(r chi.Router) {
        r.Use(a.AuthMiddleware.Authenticate)
        productH.RegisterAdminRoutes(r)
        // ...
    })
    r.Route("/api/v1/store", func(r chi.Router) {
        r.Use(a.AuthMiddleware.OptionalAuth)
        productH.RegisterStoreRoutes(r)
        // ...
    })
}
```

## Database & Migrations

- **PostgreSQL 16+**, pgxpool, golang-migrate
- Extensions: `uuid-ossp`, `pg_trgm`
- Migration naming: `000001_init.up.sql`, `000002_cart_item_upsert.up.sql`, etc.
- Always provide both `.up.sql` and `.down.sql`
- Use `CREATE UNIQUE INDEX IF NOT EXISTS` for idempotent constraints
- `WHERE ... IS NOT NULL` for partial unique indexes

## Error Handling Convention

```
Repository:  if errors.Is(err, pgx.ErrNoRows) { return nil, ErrNotFound }
             return nil, fmt.Errorf("findByID: %w", err)

Service:     if errors.Is(err, ErrNotFound) { return nil, ErrNotFound }
             return nil, fmt.Errorf("service GetByID: %w", err)

Handler:     if errors.Is(err, ErrNotFound) { h.notFound(w, "not found"); return }
             h.serverError(w, r, err)  // Logs full error, returns generic 500
```

Never expose internal errors to API consumers. Log the full error with `logger.Error().Err(err)`, respond with a generic message.

## Response Envelope

```json
{
  "data": { ... },
  "meta": { "total": 100, "page": 1, "limit": 25, "pages": 4 },
  "errors": [{ "code": "validation_error", "detail": "...", "field": "sku" }]
}
```

## Testing Conventions

- stdlib `testing`, no external framework
- Tests in the same package (`package product`, not `package product_test`)
- Mock pattern: structs with optional function fields (default = sentinel error)
- Chi URL params injected via `chi.NewRouteContext()`
- Use `httptest.NewRecorder()` and `httptest.NewRequest()` for handler tests

## Security Checklist (for core development)

1. **SQL injection**: always parameterized queries, allowlisted sort columns
2. **Auth on store routes**: `/api/v1/store/*` uses `OptionalAuth`; `/api/v1/admin/*` uses `Authenticate` + `RequireRole`
3. **IDOR prevention**: store endpoints filter by `customer_id` from auth context
4. **Error sanitization**: never leak DB errors, internal paths, or stack traces to API consumers
5. **Body size limits**: use `io.LimitReader` for user-submitted bodies
6. **CSRF**: Double Submit Cookie — exempt for Bearer/ApiKey auth
7. **Webhook handlers**: verify signatures, use background context for goroutines, implement idempotency
8. **Plugin isolation**: ScopedMCPServer (tool prefix), StoreScopedClient (path restriction), panic recovery
9. **Context propagation**: always pass `ctx` through layers; use `context.Background()` with timeout for detached goroutines
10. **Plugin UI extensions**: Tag name prefix `stoa-{pluginName}-`, URL path traversal prevention, closed Shadow DOM, SRI verification, scoped plugin API client, dynamic CSP

## Adding a New Domain

Use the `/stoa-new-domain` skill. It generates all 6 files, wires DI in `app.go`, and creates the migration.

## Adding a New Plugin

Use the `/stoa-plugin-developer` skill. Add the short name to `KnownPlugins` in `internal/plugin/installer.go`.

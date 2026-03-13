# CLAUDE.md — Stoa

Headless E-Commerce Platform — Go Backend, REST API, PostgreSQL, Plugin System. Single Binary, MCP for Agents

## Build & Development

```bash
make build                    # Build binary (admin + storefront + Go → ./stoa)
make run                      # Build + start HTTP server
go run ./cmd/stoa serve       # Start without building
go run ./cmd/stoa migrate up  # Run DB migrations
go run ./cmd/stoa seed --demo # Load demo data

make test                     # go test ./internal/... -v
make test-race                # Race detector
make lint                     # golangci-lint + go vet

make admin-dev                # Vite dev server :5173 (proxy → :8080)
make storefront-dev           # Vite dev server :5174 (proxy → :8080)

docker compose up -d postgres # Start PostgreSQL locally
```

## Module Path

`github.com/stoa-hq/stoa`

## Architecture

```
cmd/stoa/                      → Cobra CLI
cmd/stoa-store-mcp/            → Store MCP Server (16 tools)
cmd/stoa-admin-mcp/            → Admin MCP Server (33 tools)
internal/app/app.go            → DI container (Repos/Services/Handlers/Routes)
internal/server/               → HTTP server + middleware chain
internal/auth/                 → JWT, API keys, RBAC, Argon2id
internal/domain/<entity>/      → 12 domain packages (6-file pattern)
internal/mcp/                  → MCP infrastructure + store/admin tools
internal/plugin/               → Plugin registry + hook dispatch
internal/search/               → PG full-text search
internal/media/                → Local/S3 storage + image processor
internal/settings/             → Read-only config endpoint (i18n settings)
internal/config/               → Viper config
pkg/sdk/                       → Public plugin SDK
admin/                         → SvelteKit 5 Admin SPA → internal/admin/build/
storefront/                    → SvelteKit 5 Storefront SPA → internal/storefront/build/
```

**12 Domains**: product, category, customer, order, cart, tax, shipping, payment, discount, tag, audit, media

Each domain package: `entity.go`, `repository.go`, `postgres.go`, `service.go`, `handler.go`, `dto.go`
→ Details & templates: `/stoa-new-domain`

## Key Patterns

- **Prices as integers**: cents (1999 = €19.99), tax rates in basis points (1900 = 19.00%)
- **i18n (entities)**: Translatable entities have `*_translations` tables with `(entity_id, locale)` composite key
- **i18n (frontend)**: `svelte-i18n` with JSON dictionaries in `{admin,storefront}/src/lib/i18n/`. Keys namespaced by domain (e.g. `products.title`). Locale-aware formatting via `$fmt` store from `formatters.ts`. Language stored in `localStorage` (`stoa_admin_locale` / `storefront_locale`)
- **Custom fields**: Every entity has `custom_fields JSONB` (user-facing) + `metadata JSONB` (internal)
- **Response format**: `{"data": ..., "meta": {"total", "page", "limit", "pages"}, "errors": [{"code", "detail", "field"}]}`
- **Query conventions**: `?page=1&limit=25&sort=created_at&order=desc&filter[active]=true`
- **Handler response helpers**: Local per handler (`apiResponse`, `apiError`, `writeJSON`, `writeError`) — no shared package
- **Two API surfaces**: `/api/v1/admin/*` (JWT, full access) and `/api/v1/store/*` (OptionalAuth)

## Auth

- JWT Access/Refresh + Argon2id password hashing
- RBAC: `super_admin`, `admin`, `manager`, `customer`, `api_client`
- CSRF: Double Submit Cookie Pattern — requests with `Authorization` header are exempt
- JWT claims: `uid`, `email`, `utype` (admin|customer), `role`, `type` (access|refresh)
- Context helpers: `auth.UserID(ctx)`, `auth.UserType(ctx)`, `auth.UserRole(ctx)`

## Frontend (SvelteKit)

Both SPAs: SPA mode (`ssr = false`), adapter-static, Vite proxy `/api` → `:8080`

- **Admin** (`/admin`): Auth token in `localStorage` (`stoa_access_token`, `stoa_refresh_token`)
- **Storefront** (`/`): Auth token (`storefront_access_token`), cart ID (`storefront_cart_id`), locale (`storefront_locale`)
- **CSRF**: POST/PUT/PATCH/DELETE without Bearer require `X-CSRF-Token` header (cookie value `csrf_token`)
- **JWT Base64url**: `atob()` needs standard Base64 — `-`→`+`, `_`→`/`, add padding

## Test Conventions

- stdlib `testing`, no external framework
- Mock pattern: structs with optional function fields, default = sentinel error
- Tests in the same package (`package product`, not `package product_test`)
- Chi URL params injected via `chi.NewRouteContext()`
→ Details & patterns: `/stoa-test`

## Config & DB

- Config via Viper: `config.yaml` → ENV (`STOA_` prefix). Defaults in `internal/config/config.go`
- PostgreSQL 16+, pgxpool, golang-migrate, extensions: `uuid-ossp`, `pg_trgm`
- Single migration: `migrations/000001_init.up.sql` (~300 lines SQL)

## Plugin System

Plugins implement `sdk.Plugin`, receive `AppContext` (DB pool, chi sub-router, hook registry, auth helper).
Hooks: `entity.before_action` / `entity.after_action` — before-hooks can abort operations.

### Plugin Security Rules

- **Auth on store routes**: Plugin router is the ROOT Chi router — it does NOT inherit `/api/v1/store/*` middleware. Always apply `app.Auth.Required` or `app.Auth.OptionalAuth` explicitly.
- **Ownership checks**: Store-facing endpoints must verify `customer_id` matches the authenticated user (IDOR prevention).
- **MCP tool names**: Must use prefix `store_{pluginName}_*` — enforced by `ScopedMCPServer` in `internal/mcp/scoped.go`.
- **MCP client paths**: `StoreAPIClient` is restricted to `/api/v1/store/*` — enforced by `StoreScopedClient` in `internal/mcp/store_client.go`.
- **MCP type assertion**: Plugins must use interface assertion `srv.(toolAdder)` — not `srv.(*server.MCPServer)`.
- **MCP error sanitization**: Return generic errors to MCP consumers — never `err.Error()` directly.
- **Webhook idempotency**: Use `ON CONFLICT DO NOTHING` on `provider_reference` to handle duplicate deliveries.
- **Webhook goroutines**: Use `context.Background()` with timeout — not `r.Context()` which is canceled after handler returns.
- **Panic recovery**: Plugin MCP registration is wrapped in `recover()` — a buggy plugin won't crash the server.

## Skills

| Skill | Description |
|-------|-------------|
| `/stoa-new-domain` | New domain package: 6-file templates, DI wiring, migration |
| `/stoa-routes` | All API routes + MCP server (tools, config, Claude integration) |
| `/stoa-test` | Test patterns: mocks, handler tests, Chi URL params |
| `/stoa-plugin-developer` | Develops plugins for Stoa |
| `/stoa-core-developer` | Core development: domain packages, DI, auth, MCP infra, security |

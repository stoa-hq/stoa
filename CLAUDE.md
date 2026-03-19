# CLAUDE.md — Stoa

Headless E-Commerce Platform — Go, REST API, PostgreSQL, Plugin System, Single Binary, MCP for Agents

## Agent Mode

Claude MUST delegate all non-trivial work to specialized agents. Use `/stoa-core-developer` for backend, `/stoa-plugin-developer` for plugins, `/stoa-test` for tests, `/stoa-routes` for API/MCP, `/stoa-new-domain` for new domains. **Documentation MUST be delegated to the `docs-writer` agent** (not written inline). Research before implementing via `/stoa-research` or the `research` subagent. Use `Explore` agents for codebase searches. Parallelize independent agents. Only act directly for trivial edits.

## Build

```bash
make build                    # Binary (admin + storefront + Go → ./stoa)
make run                      # Build + serve
make test                     # go test ./internal/... -v
make lint                     # golangci-lint + go vet
make admin-dev                # Vite :5173 (proxy → :8080)
make storefront-dev           # Vite :5174 (proxy → :8080)
go run ./cmd/stoa migrate up  # Migrations
go run ./cmd/stoa seed --demo # Demo data
```

## Module

`github.com/stoa-hq/stoa`

## Architecture

```
cmd/stoa/                      → Cobra CLI (serve, migrate, admin, seed, plugin, version)
cmd/stoa-store-mcp/            → Store MCP Server
cmd/stoa-admin-mcp/            → Admin MCP Server
internal/app/app.go            → DI container
internal/server/               → HTTP server + middleware (CSRF, rate limit, max bytes)
internal/auth/                 → JWT, API keys, RBAC, Argon2id, brute-force, token blacklist/store
internal/domain/<entity>/      → 14 domain packages (6-file pattern)
internal/mcp/                  → MCP infra + scoped plugin enforcement
internal/plugin/               → Registry, hooks, installer, manifest handler
internal/search/               → PG full-text search
internal/media/                → Local/S3 storage + image processor
internal/settings/             → Store settings (domain-like, 6 files)
internal/crypto/               → AES encryption (payment data)
internal/csp/                  → Content Security Policy
internal/config/               → Viper config
pkg/sdk/                       → Public plugin SDK (Plugin, UIPlugin, MCPStorePlugin, HookRegistry)
admin/                         → SvelteKit 5 Admin SPA
storefront/                    → SvelteKit 5 Storefront SPA
```

**14 Domains**: product, category, customer, order, cart, tax, shipping, payment, discount, tag, audit, media, warehouse, settings

Each domain: `entity.go`, `repository.go`, `postgres.go`, `service.go`, `handler.go`, `dto.go` → `/stoa-new-domain`

## Key Patterns

- **Prices**: cents (1999 = 19.99 EUR), tax rates in basis points (1900 = 19.00%)
- **i18n**: `*_translations` tables with `(entity_id, locale)` key; frontend: `svelte-i18n` + `$fmt` store
- **Custom fields**: `custom_fields JSONB` (user) + `metadata JSONB` (internal) on every entity
- **Response**: `{"data": ..., "meta": {"total","page","limit","pages"}, "errors": [{"code","detail","field"}]}`
- **Queries**: `?page=1&limit=25&sort=created_at&order=desc&filter[active]=true`
- **Handlers**: local response helpers per handler — no shared package
- **Two APIs**: `/api/v1/admin/*` (JWT required) and `/api/v1/store/*` (OptionalAuth)

## Auth

- JWT Access/Refresh + Argon2id; RBAC: `super_admin`, `admin`, `manager`, `customer`, `api_client`
- CSRF: Double Submit Cookie — exempt with `Authorization` header
- Claims: `uid`, `email`, `utype` (admin|customer), `role`, `type` (access|refresh)
- Context: `auth.UserID(ctx)`, `auth.UserType(ctx)`, `auth.UserRole(ctx)`

## Frontend

SvelteKit 5 SPAs, `ssr=false`, adapter-static, Vite proxy `/api` → `:8080`

- **Admin** (`/admin`): localStorage keys `stoa_access_token`, `stoa_refresh_token`
- **Storefront** (`/`): keys `storefront_access_token`, `storefront_cart_id`, `storefront_locale`
- **CSRF**: non-Bearer POST/PUT/PATCH/DELETE need `X-CSRF-Token` header
- **JWT Base64url**: `atob()` needs `-`→`+`, `_`→`/`, add padding

## Tests

stdlib `testing`, mocks with optional func fields, same-package tests, Chi params via `chi.NewRouteContext()` → `/stoa-test`

## DB

PostgreSQL 16+, pgxpool, golang-migrate, 10 migrations. Extensions: `uuid-ossp`, `pg_trgm`. Config: Viper `config.yaml` → ENV `STOA_` prefix.

## Plugin System

Plugins implement `sdk.Plugin`, receive `AppContext`. Optional: `sdk.UIPlugin` (UI extensions), `sdk.MCPStorePlugin` (MCP tools).
Hooks: `entity.before_action` / `entity.after_action` — before-hooks can abort.

### Plugin Security (MUST follow)

- Plugin router = ROOT Chi router — always apply auth middleware explicitly
- Store endpoints: verify `customer_id` ownership (IDOR); guests: verify `guest_token`
- MCP: tool prefix `store_{pluginName}_*`, client restricted to `/api/v1/store/*`, use `srv.(toolAdder)` assertion, sanitize errors
- Webhooks: `ON CONFLICT DO NOTHING` on `provider_reference`; goroutines use `context.Background()` with timeout
- UI: tag prefix `stoa-{pluginName}-`, no `..` in URLs, Light DOM + scoped CSS, SRI verification
- Dynamic CSP: plugin `ExternalScripts` added to `script-src`, `frame-src`, `connect-src`
- Panic recovery wraps plugin MCP registration

## Skills

| Skill | Use for |
|-------|---------|
| `/stoa-core-developer` | Domain packages, DI, auth, MCP infra, security |
| `/stoa-plugin-developer` | Plugin development |
| `/stoa-new-domain` | New domain: 6-file templates, DI wiring, migration |
| `/stoa-routes` | API routes + MCP tools |
| `/stoa-test` | Test patterns, mocks, handler tests |
| `/stoa-docs` | Documentation (VitePress, stoa-docs repo) |
| `/stoa-research` | Codebase + docs search before implementing |
